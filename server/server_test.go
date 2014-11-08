package main

import (
	"bytes"
	"code.google.com/p/gogoprotobuf/io"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"crypto/sha256"
	"fmt"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"
)

var _ = fmt.Printf

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
	server, err := StartServer(db, shutdown)
	if err != nil {
		t.Error(err)
	}

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
	server.StopServer()

	iter := db.NewIterator(nil, nil)
	defer iter.Release()
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

//Test message listing
func TestMessageListing(t *testing.T) {
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
	server, err := StartServer(db, shutdown)
	if err != nil {
		t.Error(err)
	}
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
	envelope1Hash := sha256.Sum256([]byte("Envelope"))

	message2 := &proto.ClientToServer_DeliverEnvelope{
		User:     (*proto.Byte32)(&user),
		Envelope: []byte("Envelope2"),
	}
	deliverCommand2 := &proto.ClientToServer{
		DeliverEnvelope: message2,
	}
	err = writer.WriteMsg(deliverCommand2)
	if err != nil {
		t.Error(err)
	}
	handleResponse(conn, t)
	envelope2Hash := sha256.Sum256([]byte("Envelope2"))

	listMessages := &proto.ClientToServer{
		ListMessages: protobuf.Bool(true),
	}
	err = writer.WriteMsg(listMessages)
	if err != nil {
		t.Error(err)
	}
	response := new(proto.ServerToClient)

	reader := io.NewDelimitedReader(conn, 16*100) //TODO: what should this buffer size be?
	conn.SetDeadline(time.Now().Add(time.Second))
	if err := reader.ReadMsg(response); err != nil {
		t.Error(err)
	}
	if *response.Status == proto.ServerToClient_PARSE_ERROR {
		t.Error("Server failed to get message list.")
	}
	messageList := response.MessageList
	expected := make([][]byte, 0, 64)
	expected = append(expected, envelope1Hash[:])
	expected = append(expected, envelope2Hash[:])

	for _, hash := range expected {
		if !containsByteSlice(messageList, hash) {
			//fmt.Printf("Returned %v\n", messageList[i])
			t.Error("Wrong message list returned")
		}
	}
	server.StopServer()
}

func containsByteSlice(arr [][]byte, element []byte) bool {
	for _, arrElement := range arr {
		if bytes.Equal(arrElement, element) {
			return true
		}
	}
	return false
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
	server, err := StartServer(db, shutdown)
	if err != nil {
		t.Error(err)
	}
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
	server.StopServer()
	envelopeHash := sha256.Sum256([]byte("Envelope"))
	expectedKey := append(append([]byte{'m'}, user[:]...), envelopeHash[:]...)
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		key := iter.Key()

		if bytes.Equal(key, expectedKey) {
			return
		}
	}
	t.Error("Expected message entry not found")
}
