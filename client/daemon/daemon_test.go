package daemon

import (
	"github.com/andres-erbsen/chatterbox/client"
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
		inBuf:        make([]byte, client.MAX_MESSAGE_SIZE),
		outBuf:       make([]byte, client.MAX_MESSAGE_SIZE),
		ourDename:    []byte(alice),
	})

	bobConf := LoadConfig(&Config{
		RootDir:      dir,
		Now:          time.Now,
		TempPrefix:   "daemon",
		denameClient: bobDnmc,
		inBuf:        make([]byte, client.MAX_MESSAGE_SIZE),
		outBuf:       make([]byte, client.MAX_MESSAGE_SIZE),
		ourDename:    []byte(bob),
	})

	aliceHomeConn, alicePk, aliceSk := client.CreateTestAccount([]byte(alice), aliceDnmc, &aliceConf.LocalAccountConfig, serverAddr, serverPubkey, t)
	bobHomeConn, bobPk, bobSk := client.CreateTestAccount([]byte(bob), bobDnmc, &bobConf.LocalAccountConfig, serverAddr, serverPubkey, t)

	if err := InitFs(aliceConf); err != nil {
		t.Fatal(err)
	}

	if err := InitFs(bobConf); err != nil {
		t.Fatal(err)
	}
	aliceHomeConn = aliceHomeConn
	bobHomeConn = bobHomeConn
	alicePk = alicePk
	aliceSk = aliceSk
	bobSk = bobSk
	bobPk = bobPk
}
