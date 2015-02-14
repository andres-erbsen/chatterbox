package daemon

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	util "github.com/andres-erbsen/chatterbox/client"
	"github.com/andres-erbsen/chatterbox/client/persistence"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/chatterbox/server"
	"github.com/andres-erbsen/chatterbox/shred"
	denameClient "github.com/andres-erbsen/dename/client"
	denameTestutil "github.com/andres-erbsen/dename/testutil"
)

func TestEncryptFirstMessage(t *testing.T) {
	alice := "alice"
	bob := "bob"

	denameConfig, denameTeardown := denameTestutil.SingleServer(t)
	// FIXME: make denameTestutil.SingleServer wait until the server is up
	defer denameTeardown()

	aliceDnmc, err := denameClient.NewClient(denameConfig, nil, nil)
	bobDnmc, err := denameClient.NewClient(denameConfig, nil, nil)

	if err != nil {
		t.Fatal(err)
	}

	_, serverPubkey, serverAddr, serverTeardown := server.CreateTestServer(t)
	defer serverTeardown()

	aliceDir, err := ioutil.TempDir("", "daemon-alice")
	if err != nil {
		t.Fatal(err)
	}
	defer shred.RemoveAll(aliceDir)
	bobDir, err := ioutil.TempDir("", "daemon-alice")
	if err != nil {
		t.Fatal(err)
	}
	defer shred.RemoveAll(bobDir)

	aliceConf := &Daemon{
		Paths: persistence.Paths{
			RootDir:     aliceDir,
			Application: "daemon",
		},
		Now:                  time.Now,
		foreignDenameClient:  aliceDnmc,
		timelessDenameClient: aliceDnmc,
		inBuf:                make([]byte, proto.SERVER_MESSAGE_SIZE),
		outBuf:               make([]byte, proto.SERVER_MESSAGE_SIZE),
		LocalAccountConfig: proto.LocalAccountConfig{
			Dename: alice,
		},
	}

	bobConf := &Daemon{
		Paths: persistence.Paths{
			RootDir:     bobDir,
			Application: "daemon",
		},
		Now:                  time.Now,
		foreignDenameClient:  bobDnmc,
		timelessDenameClient: bobDnmc,
		inBuf:                make([]byte, proto.SERVER_MESSAGE_SIZE),
		outBuf:               make([]byte, proto.SERVER_MESSAGE_SIZE),
		LocalAccountConfig: proto.LocalAccountConfig{
			Dename: bob,
		},
	}

	aliceHomeConn := util.CreateTestAccount(alice, aliceDnmc, &aliceConf.LocalAccountConfig, serverAddr, serverPubkey, t)
	defer aliceHomeConn.Close()
	bobHomeConn := util.CreateTestAccount(bob, bobDnmc, &bobConf.LocalAccountConfig, serverAddr, serverPubkey, t)
	defer bobHomeConn.Close()

	//fmt.Printf("CBob: %v\n", ([32]byte)(bobConf.TransportSecretKeyForServer))
	aliceNotifies := make(chan []byte)
	aliceReplies := make(chan *proto.ServerToClient)

	aliceConnToServer := &util.ConnectionToServer{
		InBuf:        aliceConf.inBuf,
		Conn:         aliceHomeConn,
		ReadReply:    aliceReplies,
		ReadEnvelope: aliceNotifies,
	}

	go aliceConnToServer.ReceiveMessages()

	bobNotifies := make(chan []byte)
	bobReplies := make(chan *proto.ServerToClient)

	bobConnToServer := &util.ConnectionToServer{
		InBuf:        bobConf.inBuf,
		Conn:         bobHomeConn,
		ReadReply:    bobReplies,
		ReadEnvelope: bobNotifies,
	}

	go bobConnToServer.ReceiveMessages()

	if err := InitFs(aliceConf); err != nil {
		t.Fatal(err)
	}

	if err := InitFs(bobConf); err != nil {
		t.Fatal(err)
	}
	//Bob uploads keys
	bobPublicPrekeys, bobSecretPrekeys, err := GeneratePrekeys(maxPrekeys)
	var bobSigningKey [64]byte
	copy(bobSigningKey[:], bobConf.KeySigningSecretKey[:64])
	err = util.UploadKeys(bobConnToServer, util.SignKeys(bobPublicPrekeys, &bobSigningKey))
	if err != nil {
		t.Fatal(err)
	}

	//Bob enables notifications
	if err = util.EnablePush(bobConnToServer); err != nil {
		t.Fatal(err)
	}

	//Alice uploads keys
	alicePublicPrekeys, _, err := GeneratePrekeys(maxPrekeys)
	var aliceSigningKey [64]byte
	copy(aliceSigningKey[:], aliceConf.KeySigningSecretKey[:64])
	err = util.UploadKeys(aliceConnToServer, util.SignKeys(alicePublicPrekeys, &aliceSigningKey))
	if err != nil {
		t.Fatal(err)
	}

	//Alice enables notification
	if err = util.EnablePush(aliceConnToServer); err != nil {
		t.Fatal(err)
	}

	participants := make([]string, 0, 2)
	participants = append(participants, alice)
	participants = append(participants, bob)

	msg1 := []byte("Envelope")

	payload := proto.Message{
		Subject:      "Subject1",
		Participants: participants,
		Dename:       alice,
		Contents:     msg1,
	}
	envelope, err := payload.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	aliceRatch, err := aliceConf.sendFirstMessage(envelope, bob)
	if err != nil {
		t.Fatal(err)
	}
	incoming := <-bobConnToServer.ReadEnvelope

	out, bobRatch, _, err := bobConf.decryptFirstMessage(incoming, bobPublicPrekeys, bobSecretPrekeys)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Bob hears: %s\n", out)

	msg2 := []byte("Envelope2")
	payload2 := proto.Message{
		Subject:      "Subject3",
		Participants: participants,
		Dename:       bob,
		Contents:     msg2,
	}
	envelope2, err := payload2.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	bobRatch, err = bobConf.sendMessage(envelope2, alice, bobRatch)
	if err != nil {
		t.Fatal(err)
	}

	incomingAlice := <-aliceConnToServer.ReadEnvelope

	aliceRatchets := make([]*ratchet.Ratchet, 0)
	aliceRatchets = append(aliceRatchets, aliceRatch)
	outAlice, aliceRatch, err := aliceConf.decryptMessage(incomingAlice, aliceRatchets)
	if err != nil {
		t.Fatal(err)
	}

	ha := sha256.Sum256(incomingAlice)
	if err = aliceConf.receiveMessage(aliceConnToServer, outAlice, &ha); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Alice hears: %s\n", outAlice)

	//TODO: Confirm message is as expected within the test
}
