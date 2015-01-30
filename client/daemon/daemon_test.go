package daemon

import (
	"fmt"
	util "github.com/andres-erbsen/chatterbox/client"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/chatterbox/server"
	"github.com/andres-erbsen/chatterbox/shred"
	denameClient "github.com/andres-erbsen/dename/client"
	denameTestutil "github.com/andres-erbsen/dename/testutil"
	"io/ioutil"
	"testing"
	"time"
)

func TestEncryptFirstMessage(t *testing.T) {
	t.Skip("apparently we can only run a single test involving a dename server. this may have something to do with TCP TIME_WAIT. I (andres) am disabling this test to make the other one pass (this one looks more sketchy).")
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

	// XXX: should alice and bob have separate directories? or do they already?
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer shred.RemoveAll(dir)

	aliceConf := &Config{
		RootDir:      dir,
		Now:          time.Now,
		TempPrefix:   "daemon",
		denameClient: aliceDnmc,
		inBuf:        make([]byte, proto.SERVER_MESSAGE_SIZE),
		outBuf:       make([]byte, proto.SERVER_MESSAGE_SIZE),
		LocalAccountConfig: proto.LocalAccountConfig{
			Dename: alice,
		},
	}

	bobConf := &Config{
		RootDir:      dir,
		Now:          time.Now,
		TempPrefix:   "daemon",
		denameClient: bobDnmc,
		inBuf:        make([]byte, proto.SERVER_MESSAGE_SIZE),
		outBuf:       make([]byte, proto.SERVER_MESSAGE_SIZE),
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
		Buf:          aliceConf.inBuf,
		Conn:         aliceHomeConn,
		ReadReply:    aliceReplies,
		ReadEnvelope: aliceNotifies,
	}

	go aliceConnToServer.ReceiveMessages()

	bobNotifies := make(chan []byte)
	bobReplies := make(chan *proto.ServerToClient)

	bobConnToServer := &util.ConnectionToServer{
		Buf:          bobConf.inBuf,
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
	bobPublicPrekeys, bobSecretPrekeys, err := GeneratePrekeys(MAX_PREKEYS)
	var bobSigningKey [64]byte
	copy(bobSigningKey[:], bobConf.KeySigningSecretKey[:64])
	err = util.UploadKeys(bobHomeConn, bobConnToServer, bobConf.outBuf, util.SignKeys(bobPublicPrekeys, &bobSigningKey))
	if err != nil {
		t.Fatal(err)
	}

	//Bob enables notifications
	if err = util.EnablePush(bobHomeConn, bobConnToServer, bobConf.outBuf); err != nil {
		t.Fatal(err)
	}

	//Alice uploads keys
	alicePublicPrekeys, _, err := GeneratePrekeys(MAX_PREKEYS)
	var aliceSigningKey [64]byte
	copy(aliceSigningKey[:], aliceConf.KeySigningSecretKey[:64])
	err = util.UploadKeys(aliceHomeConn, aliceConnToServer, aliceConf.outBuf, util.SignKeys(alicePublicPrekeys, &aliceSigningKey))
	if err != nil {
		t.Fatal(err)
	}

	//Alice enables notification
	if err = util.EnablePush(aliceHomeConn, aliceConnToServer, aliceConf.outBuf); err != nil {
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

	fmt.Printf("Alice hears: %s\n", outAlice)

	//TODO: Confirm message is as expected within the test
}
