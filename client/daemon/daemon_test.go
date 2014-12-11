package daemon

import (
	"fmt"
	util "github.com/andres-erbsen/chatterbox/client"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	//protobuf "code.google.com/p/gogoprotobuf/proto"
	"github.com/andres-erbsen/chatterbox/server"
	denameClient "github.com/andres-erbsen/dename/client"
	denameTestutil "github.com/andres-erbsen/dename/testutil"
	"io/ioutil"
	"os"
	"testing"
	"time"
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
	time.Sleep(1)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	aliceConf := &Config{
		RootDir:      dir,
		Now:          time.Now,
		TempPrefix:   "daemon",
		denameClient: aliceDnmc,
		inBuf:        make([]byte, util.MAX_MESSAGE_SIZE),
		outBuf:       make([]byte, util.MAX_MESSAGE_SIZE),
		LocalAccountConfig: proto.LocalAccountConfig{
			Dename: []byte(alice),
		},
	}

	bobConf := &Config{
		RootDir:      dir,
		Now:          time.Now,
		TempPrefix:   "daemon",
		denameClient: bobDnmc,
		inBuf:        make([]byte, util.MAX_MESSAGE_SIZE),
		outBuf:       make([]byte, util.MAX_MESSAGE_SIZE),
		LocalAccountConfig: proto.LocalAccountConfig{
			Dename: []byte(bob),
		},
	}

	aliceHomeConn := util.CreateTestAccount([]byte(alice), aliceDnmc, &aliceConf.LocalAccountConfig, serverAddr, serverPubkey, t)
	bobHomeConn := util.CreateTestAccount([]byte(bob), bobDnmc, &bobConf.LocalAccountConfig, serverAddr, serverPubkey, t)

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

	envelope := []byte("Envelope")

	aliceRatch, err := aliceConf.sendFirstMessage(envelope, []byte(bob))
	if err != nil {
		t.Fatal(err)
	}

	incoming := <-bobConnToServer.ReadEnvelope

	out, bobRatch, _, err := bobConf.decryptFirstMessage(incoming, bobPublicPrekeys, bobSecretPrekeys)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Bob hears: %s\n", out)

	envelope2 := []byte("Envelope2")
	bobRatch, err = bobConf.sendMessage(envelope2, []byte(alice), bobRatch)
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
}
