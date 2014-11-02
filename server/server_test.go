package main

import (
	//	"bytes"
	"code.google.com/p/gogoprotobuf/io"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
	"net"
	"os"
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
	command := &proto.ClientToServer{
		CreateAccount: protobuf.Bool(true),
	}
	err = writer.WriteMsg(command)
	if err != nil {
		t.Error(err)
	}
	handleResponse(conn, t)
	close(shutdown)
	//	userIter := db.NewIterator(nil, nil)
	//	userIter.First()
	//	user := [32]byte{}
	//	copy(user[:], userIter.Key()[1:32])
	//	message := &proto.ClientToServer_DeliverEnvelope{
	//		User:     (*proto.Byte32)(&user),
	//		Envelope: []byte("Envelope"),
	//	}
	//	deliverCommand := &proto.ClientToServer{
	//		DeliverEnvelope: message,
	//	}
	//	err = writer.WriteMsg(deliverCommand)
	//	if err != nil {
	//		t.Error(err)
	//	}
	//	handleResponse(conn, t)
	//	close(shutdown)
	//	envelopeHash := []byte("47a1436c7090cfb59614a6bee2c6ef99043cddceda2b6df9d996427f6e42077d")
	//	expectedKey := append(append([]byte{'u'}, user[:]...), envelopeHash[:]...)
	//	iter := db.NewIterator(nil, nil)
	//	for iter.Next() {
	//		key := iter.Key()
	//		if bytes.Equal(key, expectedKey) {
	//			return
	//		}
	//	}
	//	t.Error("Expected message entry not found")
}
