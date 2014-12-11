package server

import (
	"code.google.com/p/go.crypto/nacl/box"
	"crypto/rand"
	"github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
	"os"
	"testing"
)

func CreateTestServer(t *testing.T) (*Server, *[32]byte, func()) {
	dir, err := ioutil.TempDir("", "testdb")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	shutdown := make(chan struct{})

	pks, sks, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	server, err := StartServer(db, shutdown, pks, sks, ":0")
	if err != nil {
		t.Fatal(err)
	}

	return server, pks, func() {
		server.StopServer()
		os.RemoveAll(dir)
		db.Close()
	}
}
