package main

import (
	"bytes"
	"code.google.com/p/gogoprotobuf/io"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"crypto/sha256"
	//	"fmt"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

//Tests whether database contains new account after creating one
func TestAccountCreation(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	if err != nil {
		t.Error(err)
	}

	defer db.Close()
	shutdown := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { RunServer(db, shutdown); wg.Done() }()

	conn, err := net.Dial("tcp", "localhost:8888")
	if err != nil {
		t.Error(err)
	}
	defer conn.Close()
	writer := io.NewDelimitedWriter(conn)
	defer writer.Close()
	command := &proto.ClientToServer{
		CreateAccount: protobuf.Bool(true),
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
	wg.Wait()
}

func handleResponse(connection net.Conn, t *testing.T) error {
	response := new(proto.ServerToClient)

	reader := io.NewDelimitedReader(connection, 16*100) //TODO: what should this buffer size be?
	connection.SetDeadline(time.Now().Add(time.Second))
	if err := reader.ReadMsg(response); err != nil {
		t.Error(err)
	}
	if *response.Status == proto.ServerToClient_PARSE_ERROR {
		t.Error("Server failed to update database.")
	}
	return nil //We did it!
}

// Tests whether database contains new message after uploading one
func TestMessageUploading(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	shutdown := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { RunServer(db, shutdown); wg.Done() }()
	conn, err := net.Dial("tcp", "localhost:8888")
	if err != nil {
		t.Error(err)
	}
	defer conn.Close()
	writer := io.NewDelimitedWriter(conn)
	command := &proto.ClientToServer{
		CreateAccount: protobuf.Bool(true),
	}
	err = writer.WriteMsg(command)
	if err != nil {
		t.Error(err)
	}
	handleResponse(conn, t)
	userIter := db.NewIterator(nil, nil)
	userIter.First()
	user := [32]byte{}
	copy(user[:], userIter.Key()[1:32])
	message := &proto.ClientToServer_DeliverEnvelope{
		User:     (*proto.Byte32)(&user),
		Envelope: []byte("Envelope"),
	}
	deliverCommand := &proto.ClientToServer{
		DeliverEnvelope: message,
	}
	err = writer.WriteMsg(deliverCommand)
	if err != nil {
		t.Error(err)
	}
	handleResponse(conn, t)
	close(shutdown)
	envelopeHash := sha256.Sum256([]byte("Envelope"))
	expectedKey := append(append([]byte{'m'}, user[:]...), envelopeHash[:]...)
	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()

		if bytes.Equal(key, expectedKey) {
			return
		}
	}
	t.Error("Expected message entry not found")
	wg.Wait()
}
