package main

import (
	"code.google.com/p/gogoprotobuf/io"
	"code.google.com/p/gogoprotobuf/proto"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"
)

func TestAccountCreation(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	defer db.Close()
	if err != nil {
		t.Error(err)
	}
	shutdown := make(chan struct{})
	go RunServer(db, shutdown)
	conn, err := net.Dial("tcp", "localhost:8888")
	defer conn.Close()
	if err != nil {
		t.Error(err)
	}
	writer := io.NewDelimitedWriter(conn)
	command := &Messages.ClientToServer{
		CreateAccount: proto.Bool(true),
	}
	err = writer.WriteMsg(command)
	if err != nil {
		t.Error(err)
	}
	handleResponse(conn, t)
	close(shutdown)
	iter := db.NewIterator(nil, nil)
	if !iter.First() {
		t.Error("Nothing in database")
	}
}

func handleResponse(connection net.Conn, t *testing.T) error {
	response := new(Messages.ServerToClient)
	reader := io.NewDelimitedReader(connection, 16*100) //TODO: what should this buffer size be?
	connection.SetDeadline(time.Now().Add(time.Second))
	if err := reader.ReadMsg(response); err != nil {
		t.Error(err)
	}
	if *response.Status == Messages.ServerToClient_PARSE_ERROR {
		t.Error("Server failed to update database.")
	}
	return nil //We did it!
}
