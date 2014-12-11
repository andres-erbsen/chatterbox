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

	aliceConf := LoadConfig(&Config{
		RootDir:      dir,
		Now:          time.Now,
		TempPrefix:   "daemon",
		denameClient: aliceDnmc,
		inBuf:        make([]byte, util.MAX_MESSAGE_SIZE),
		outBuf:       make([]byte, util.MAX_MESSAGE_SIZE),
		ourDename:    []byte(alice),
	})

	bobConf := LoadConfig(&Config{
		RootDir:      dir,
		Now:          time.Now,
		TempPrefix:   "daemon",
		denameClient: bobDnmc,
		inBuf:        make([]byte, util.MAX_MESSAGE_SIZE),
		outBuf:       make([]byte, util.MAX_MESSAGE_SIZE),
		ourDename:    []byte(bob),
	})

	fmt.Printf("Address: %v\n", serverAddr)

	aliceHomeConn, alicePk, aliceSk := util.CreateTestAccount([]byte(alice), aliceDnmc, &aliceConf.LocalAccountConfig, serverAddr, serverPubkey, t)
	bobHomeConn, bobPk, bobSk := util.CreateTestAccount([]byte(bob), bobDnmc, &bobConf.LocalAccountConfig, serverAddr, serverPubkey, t)

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
	publicPrekeys, newSecretPrekeys, err := GeneratePrekeys(MAX_PREKEYS)
	var signingKey [64]byte
	copy(signingKey[:], bobConf.KeySigningSecretKey[:64])
	err = util.UploadKeys(bobHomeConn, bobConnToServer, bobConf.outBuf, util.SignKeys(publicPrekeys, &signingKey))
	if err != nil {
		t.Fatal(err)
	}

	newSecretPrekeys = newSecretPrekeys
	aliceHomeConn = aliceHomeConn
	bobHomeConn = bobHomeConn
	alicePk = alicePk
	aliceSk = aliceSk
	bobSk = bobSk
	bobPk = bobPk
}
