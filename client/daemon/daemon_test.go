package daemon

import (
	"fmt"
	util "github.com/andres-erbsen/chatterbox/client"
	"github.com/andres-erbsen/chatterbox/proto"
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
	publicPrekeys, newSecretPrekeys, err := GeneratePrekeys(1)
	var signingKey [64]byte
	copy(signingKey[:], bobConf.KeySigningSecretKey[:64])
	err = util.UploadKeys(bobHomeConn, bobConnToServer, bobConf.outBuf, util.SignKeys(publicPrekeys, &signingKey))
	if err != nil {
		t.Fatal(err)
	}

	//_, err := util.GetNumKeys(bobHomeConn, bobConnToServer, bobConf.outBuf)
	//if err != nil {
	//t.Fatal(err)
	//}
	//Alice encrypts and sends a message
	envelope := []byte("Envelope")

	err = aliceConf.encryptFirstMessage(envelope, []byte(bob))
	if err != nil {
		t.Fatal(err)
	}

	//Bob checks his messages
	messages, err := util.ListUserMessages(bobHomeConn, bobConnToServer, bobConf.outBuf)
	if err != nil {
		t.Fatal(err)
	}
	//fmt.Printf("Messages: %v\n", messages)

	err = util.RequestMessage(bobHomeConn, bobConnToServer, bobConf.outBuf, &(messages[0]))
	if err != nil {
		t.Fatal(err)
	}

	incoming := <-bobConnToServer.ReadEnvelope

	fmt.Printf("Correct PK %x\n", publicPrekeys[0])

	out, err := bobConf.decryptFirstMessage(incoming, newSecretPrekeys)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Bob hears: %s\n", out)

	newSecretPrekeys = newSecretPrekeys
	aliceHomeConn = aliceHomeConn
	bobHomeConn = bobHomeConn
}
