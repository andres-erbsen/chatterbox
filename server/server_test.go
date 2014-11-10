package main

import (
	"bytes"
	"code.google.com/p/go.crypto/nacl/box"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/transport"
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
	handleError(err, t)

	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	handleError(err, t)

	defer db.Close()
	shutdown := make(chan struct{})

	pks, sks, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	server, err := StartServer(db, shutdown, pks, sks)
	handleError(err, t)

	oldConn, err := net.Dial("tcp", "localhost:8888")
	handleError(err, t)

	pkp, skp, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	conn, _, err := transport.Handshake(oldConn, pkp, skp, nil, MAX_MESSAGE_SIZE)
	handleError(err, t)

	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	command := &proto.ClientToServer{
		CreateAccount: protobuf.Bool(true),
	}
	err = writeProtobuf(conn, outBuf, command, t)
	handleError(err, t)

	handleResponse(conn, t)

	server.StopServer()

	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	if !iter.First() {
		t.Error("Nothing in database")
	}
}

func handleError(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
	}
}

//func TestMarshalTo(t *testing.T) {
//	outBuf := make([]byte, MAX_MESSAGE_SIZE)
//	user := [32]byte{}
//	message := &proto.ClientToServer_DeliverEnvelope{
//		User:     (*proto.Byte32)(&user),
//		Envelope: []byte("Envelope"),
//	}
//	sent := &proto.ClientToServer{
//		DeliverEnvelope: message,
//	}
//	fmt.Printf("Original object\n %v\n", sent)
//	size, err := sent.MarshalTo(outBuf)
//	handleError(err, t)
//
//	received := new(proto.ClientToServer)
//
//	received.Unmarshal(outBuf[:size])
//	fmt.Printf("Unmarshalled object\n %v\n", received)
//}

func writeProtobuf(conn *transport.Conn, outBuf []byte, message *proto.ClientToServer, t *testing.T) error {
	size, err := message.MarshalTo(outBuf)
	handleError(err, t)
	conn.WriteFrame(outBuf[:size])
	return nil
}

func handleResponse(connection *transport.Conn, t *testing.T) error {
	response := new(proto.ServerToClient)

	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	connection.SetDeadline(time.Now().Add(time.Second))

	num, err := connection.ReadFrame(inBuf)
	handleError(err, t)

	if err := response.Unmarshal(inBuf[:num]); err != nil {
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
	handleError(err, t)

	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	handleError(err, t)

	defer db.Close()
	shutdown := make(chan struct{})

	pk, sk, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	server, err := StartServer(db, shutdown, pk, sk)
	handleError(err, t)

	oldConn, err := net.Dial("tcp", "localhost:8888")
	handleError(err, t)

	pkp, skp, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	conn, _, err := transport.Handshake(oldConn, pkp, skp, nil, MAX_MESSAGE_SIZE)
	handleError(err, t)

	outBuf := make([]byte, MAX_MESSAGE_SIZE)
	command := &proto.ClientToServer{
		CreateAccount: protobuf.Bool(false),
	}
	err = writeProtobuf(conn, outBuf, command, t)
	handleError(err, t)

	handleResponse(conn, t)

	message := &proto.ClientToServer_DeliverEnvelope{
		User:     (*proto.Byte32)(pkp),
		Envelope: []byte("Envelope"),
	}
	deliverCommand := &proto.ClientToServer{
		DeliverEnvelope: message,
	}
	err = writeProtobuf(conn, outBuf, deliverCommand, t)
	handleError(err, t)

	handleResponse(conn, t)

	server.StopServer()
	envelopeHash := sha256.Sum256([]byte("Envelope"))
	expectedKey := append(append([]byte{'m'}, (*pkp)[:]...), envelopeHash[:]...)
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

//Test message listing
func TestMessageListing(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	handleError(err, t)

	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	handleError(err, t)

	defer db.Close()

	shutdown := make(chan struct{})

	pk, sk, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	server, err := StartServer(db, shutdown, pk, sk)
	handleError(err, t)

	oldConn, err := net.Dial("tcp", "localhost:8888")
	handleError(err, t)

	pkp, skp, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	conn, _, err := transport.Handshake(oldConn, pkp, skp, nil, MAX_MESSAGE_SIZE)
	handleError(err, t)

	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	command := &proto.ClientToServer{
		CreateAccount: protobuf.Bool(true),
	}
	err = writeProtobuf(conn, outBuf, command, t)
	handleError(err, t)

	handleResponse(conn, t)

	message := &proto.ClientToServer_DeliverEnvelope{
		User:     (*proto.Byte32)(pkp),
		Envelope: []byte("Envelope"),
	}
	deliverCommand := &proto.ClientToServer{
		DeliverEnvelope: message,
	}
	err = writeProtobuf(conn, outBuf, deliverCommand, t)
	handleError(err, t)

	handleResponse(conn, t)

	message2 := &proto.ClientToServer_DeliverEnvelope{
		User:     (*proto.Byte32)(pkp),
		Envelope: []byte("Envelope2"),
	}
	deliverCommand2 := &proto.ClientToServer{
		DeliverEnvelope: message2,
	}
	err = writeProtobuf(conn, outBuf, deliverCommand2, t)
	handleError(err, t)

	handleResponse(conn, t)

	listMessages := &proto.ClientToServer{
		ListMessages: protobuf.Bool(true),
	}
	err = writeProtobuf(conn, outBuf, listMessages, t)

	handleError(err, t)

	response := new(proto.ServerToClient)

	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	conn.SetDeadline(time.Now().Add(time.Second))
	num, err := conn.ReadFrame(inBuf)
	handleError(err, t)
	if err := response.Unmarshal(inBuf[:num]); err != nil {
		t.Error(err)
	}

	if *response.Status == proto.ServerToClient_PARSE_ERROR {
		t.Error("Server failed to get message list.")
	}
	messageList := response.MessageList
	expected := make([][]byte, 0, 64)
	envelope1Hash := sha256.Sum256([]byte("Envelope"))
	envelope2Hash := sha256.Sum256([]byte("Envelope2"))
	expected = append(expected, envelope1Hash[:])
	expected = append(expected, envelope2Hash[:])

	for _, hash := range expected {
		if !containsByteSlice(messageList, hash) {
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
