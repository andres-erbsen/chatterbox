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
	dnmc, err := denameClient.NewClient(denameConfig, nil, nil)
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

	conf := LoadConfig(&Config{
		RootDir:      dir,
		Now:          time.Now,
		TempPrefix:   "daemon",
		denameClient: dnmc,
		inBuf:        make([]byte, client.MAX_MESSAGE_SIZE),
		outBuf:       make([]byte, client.MAX_MESSAGE_SIZE),
		ourDename:    []byte(alice),
	})

	aliceHomeConn := client.CreateTestAccount([]byte(alice), dnmc, &bobConf.LocalAccountConfig, serverAddr, serverPubkey, t)
	bobHomeConn := client.CreateTestAccount([]byte(bob), dnmc, &aliceConf.LocalAccountConfig, serverAddr, serverPubkey, t)

	if err := InitFs(conf); err != nil {
		t.Fatal(err)
	}

}
