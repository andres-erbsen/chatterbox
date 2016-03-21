package server

import (
	"bytes"
	"golang.org/x/crypto/nacl/box"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/transport"
	protobuf "github.com/gogo/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"
)

func handleError(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
	}
}

func writeProtobuf(conn *transport.Conn, outBuf []byte, message *proto.ClientToServer, t *testing.T) {
	unpadMsg, err := protobuf.Marshal(message)
	handleError(err, t)

	padMsg := proto.Pad(unpadMsg, proto.SERVER_MESSAGE_SIZE)
	copy(outBuf, padMsg)

	conn.WriteFrame(outBuf[:proto.SERVER_MESSAGE_SIZE])
}

func receiveProtobuf(conn *transport.Conn, inBuf []byte, t *testing.T) *proto.ServerToClient {
	response := new(proto.ServerToClient)
	conn.SetDeadline(time.Now().Add(time.Second))
	num, err := conn.ReadFrame(inBuf)
	handleError(err, t)
	unpadMsg := proto.Unpad(inBuf[:num])
	if err := response.Unmarshal(unpadMsg); err != nil {
		t.Error(err)
	}
	if response.Status == nil {
		t.Error("Server returned nil status.")
	}
	if *response.Status == proto.ServerToClient_PARSE_ERROR {
		t.Error("Server threw a parse error.")
	}
	return response
}

func containsByteSlice(arr [][]byte, element []byte) bool {
	for _, arrElement := range arr {
		if bytes.Equal(arrElement, element) {
			return true
		}
	}
	return false
}

func contains32Byte(arr [][32]byte, element [32]byte) bool {
	for _, arrElement := range arr {
		if arrElement == element {
			return true
		}
	}
	return false
}
func setUpServerTest(db *leveldb.DB, t *testing.T) (*Server, *transport.Conn, []byte, []byte, *[32]byte) {
	shutdown := make(chan struct{})

	pks, sks, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	server, err := StartServer(db, shutdown, pks, sks, ":0")
	handleError(err, t)

	oldConn, err := net.Dial("tcp", server.listener.Addr().String())
	handleError(err, t)

	pkp, skp, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	conn, _, err := transport.Handshake(oldConn, pkp, skp, nil, proto.SERVER_MESSAGE_SIZE)
	handleError(err, t)

	inBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)
	outBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)

	return server, conn, inBuf, outBuf, pkp
}

func createAccount(conn *transport.Conn, inBuf []byte, outBuf []byte, t *testing.T) {
	command := &proto.ClientToServer{
		CreateAccount: protobuf.Bool(true),
	}
	writeProtobuf(conn, outBuf, command, t)

	receiveProtobuf(conn, inBuf, t)
}

//Tests whether database contains new account after creating one
func TestAccountCreation(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	handleError(err, t)

	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	handleError(err, t)

	defer db.Close()

	server, conn, inBuf, outBuf, _ := setUpServerTest(db, t)
	defer conn.Close()

	createAccount(conn, inBuf, outBuf, t)

	server.StopServer()

	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	if !iter.First() {
		t.Error("Nothing in database")
	}
}

func uploadMessageToUser(conn *transport.Conn, inBuf []byte, outBuf []byte, t *testing.T, pk *[32]byte, envelope []byte) {
	message := &proto.ClientToServer_DeliverEnvelope{
		User:     (*proto.Byte32)(pk),
		Envelope: envelope,
	}
	deliverCommand := &proto.ClientToServer{
		DeliverEnvelope: message,
	}
	writeProtobuf(conn, outBuf, deliverCommand, t)

	receiveProtobuf(conn, inBuf, t)
}

// Tests whether database contains new message after uploading one
func TestMessageUploading(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	handleError(err, t)

	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	handleError(err, t)

	defer db.Close()

	server, conn, inBuf, outBuf, pkp := setUpServerTest(db, t)
	defer conn.Close()

	envelope := []byte("Envelope")

	createAccount(conn, inBuf, outBuf, t)
	uploadMessageToUser(conn, inBuf, outBuf, t, pkp, envelope)

	server.StopServer()

	envelopeHash := sha256.Sum256(envelope)
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		key := iter.Key()
		if bytes.Equal(key[1+32+8:], envelopeHash[:24]) {
			return
		}
	}
	t.Error("Expected message entry not found")
}

func listUserMessages(conn *transport.Conn, inBuf []byte, outBuf []byte, t *testing.T) []*[32]byte {
	listMessages := &proto.ClientToServer{
		ListMessages: protobuf.Bool(true),
	}
	writeProtobuf(conn, outBuf, listMessages, t)

	response := receiveProtobuf(conn, inBuf, t)

	return proto.To32ByteList(response.MessageList)
}

//Test message listing
func TestMessageListing(t *testing.T) {
	server, pks, _, teardown := CreateTestServer(t)

	oldConn, err := net.Dial("tcp", server.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer oldConn.Close()

	pkp, skp, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	conn, _, err := transport.Handshake(oldConn, pkp, skp, pks, proto.SERVER_MESSAGE_SIZE)
	if err != nil {
		t.Fatal(err)
	}

	inBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)
	outBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)

	envelope1 := []byte("Envelope1")
	envelope2 := []byte("Envelope2")

	createAccount(conn, inBuf, outBuf, t)
	uploadMessageToUser(conn, inBuf, outBuf, t, pkp, envelope1)
	uploadMessageToUser(conn, inBuf, outBuf, t, pkp, envelope2)

	messageList := listUserMessages(conn, inBuf, outBuf, t)

	expected := make([][32]byte, 0, 64)
	envelope1Hash := sha256.Sum256(envelope1)
	envelope2Hash := sha256.Sum256(envelope2)
	expected = append(expected, envelope1Hash)
	expected = append(expected, envelope2Hash)

outer:
	for _, hash := range expected {
		for _, msgid := range messageList {
			if bytes.Equal(msgid[8:], hash[:24]) {
				continue outer
			}
		}
		t.Error("Wrong message list returned")
	}
	teardown()
}

func downloadEnvelope(conn *transport.Conn, inBuf []byte, outBuf []byte, t *testing.T, messageHash *[32]byte) []byte {
	getEnvelope := &proto.ClientToServer{
		DownloadEnvelope: (*proto.Byte32)(messageHash),
	}
	writeProtobuf(conn, outBuf, getEnvelope, t)

	response := receiveProtobuf(conn, inBuf, t)
	return response.Envelope
}

//Test downloading envelopes
func TestEnvelopeDownload(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	handleError(err, t)

	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	handleError(err, t)

	defer db.Close()

	server, conn, inBuf, outBuf, pkp := setUpServerTest(db, t)
	defer conn.Close()

	envelope1 := []byte("Envelope1")
	envelope2 := []byte("Envelope2")

	createAccount(conn, inBuf, outBuf, t)
	uploadMessageToUser(conn, inBuf, outBuf, t, pkp, envelope1)
	uploadMessageToUser(conn, inBuf, outBuf, t, pkp, envelope2)

	messageList := listUserMessages(conn, inBuf, outBuf, t)

	//TODO: Should messageHash just be 32-bytes? Answer: Probably yes, oh well
	for _, msgid := range messageList {
		envelope := downloadEnvelope(conn, inBuf, outBuf, t, msgid)

		h := sha256.Sum256(envelope)
		if !bytes.Equal(msgid[8:], h[:24]) {
			t.Error("Wrong envelope associated with message")
		}
	}
	server.StopServer()
}

func deleteMessages(conn *transport.Conn, inBuf []byte, outBuf []byte, t *testing.T, messageList []*[32]byte) {
	deleteMessages := &proto.ClientToServer{
		DeleteMessages: proto.ToProtoByte32List(messageList),
	}
	writeProtobuf(conn, outBuf, deleteMessages, t)

	receiveProtobuf(conn, inBuf, t)
}

func TestListAndDelete(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	handleError(err, t)

	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	handleError(err, t)

	defer db.Close()

	server, conn, inBuf, outBuf, pkp := setUpServerTest(db, t)
	defer conn.Close()

	envelope1 := []byte("Envelope1")
	envelope2 := []byte("Envelope2")

	createAccount(conn, inBuf, outBuf, t)
	uploadMessageToUser(conn, inBuf, outBuf, t, pkp, envelope1)
	uploadMessageToUser(conn, inBuf, outBuf, t, pkp, envelope2)

	messageList := listUserMessages(conn, inBuf, outBuf, t)

	deleteMessages(conn, inBuf, outBuf, t, messageList)

	newMessageList := listUserMessages(conn, inBuf, outBuf, t)

	if !(len(newMessageList) == 0) {
		t.Error("Not all messages properly deleted")
	}

	server.StopServer()
}

func uploadKeys(conn *transport.Conn, inBuf []byte, outBuf []byte, t *testing.T, keyList [][]byte) {
	uploadKeys := &proto.ClientToServer{
		UploadSignedKeys: keyList,
	}
	writeProtobuf(conn, outBuf, uploadKeys, t)

	receiveProtobuf(conn, inBuf, t)
}

func getKey(conn *transport.Conn, inBuf []byte, outBuf []byte, t *testing.T, pk *[32]byte) []byte {
	getKey := &proto.ClientToServer{
		GetSignedKey: (*proto.Byte32)(pk),
	}
	writeProtobuf(conn, outBuf, getKey, t)

	response := receiveProtobuf(conn, inBuf, t)
	return response.SignedKey
}

func getNumKeys(conn *transport.Conn, inBuf []byte, outBuf []byte, t *testing.T, pk *[32]byte) int64 {
	getNumKeys := &proto.ClientToServer{
		GetNumKeys: protobuf.Bool(true),
	}
	writeProtobuf(conn, outBuf, getNumKeys, t)

	response := receiveProtobuf(conn, inBuf, t)
	return *response.NumKeys
}

func TestGetNumberOfKeys(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	handleError(err, t)

	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	handleError(err, t)

	defer db.Close()

	server, conn, inBuf, outBuf, pkp := setUpServerTest(db, t)
	defer conn.Close()

	createAccount(conn, inBuf, outBuf, t)

	pk1, _, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	pk2, _, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	// NOTE: the keys are note signed here, but they will be in real use
	keyList := make([][]byte, 0, 64) //TODO: Make this a reasonable size
	keyList = append(keyList, pk1[:])
	keyList = append(keyList, pk2[:])

	uploadKeys(conn, inBuf, outBuf, t, keyList)
	newKey1 := getKey(conn, inBuf, outBuf, t, pkp)

	if newKey1 == nil {
		t.Error("No keys in server")
	}
	if !(containsByteSlice(keyList, newKey1)) {
		t.Error("Non-uploaded key returned")
	}

	newKey2 := getKey(conn, inBuf, outBuf, t, pkp)
	if newKey2 == nil {
		t.Fatal("No keys in server")
	}
	if !(containsByteSlice(keyList, newKey2)) {
		t.Error("Non-uploaded key returned")
	}
	if bytes.Equal(newKey1, newKey2) {
		t.Error("Key not deleted from server")
	}

	server.StopServer()
}

func TestKeyUploadDownload(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	handleError(err, t)

	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	handleError(err, t)

	defer db.Close()

	server, conn, inBuf, outBuf, pkp := setUpServerTest(db, t)
	defer conn.Close()

	createAccount(conn, inBuf, outBuf, t)

	pk1, _, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	pk2, _, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	keyList := make([][]byte, 0, 64) //TODO: Make this a reasonable size
	keyList = append(keyList, pk1[:])
	keyList = append(keyList, pk2[:])

	uploadKeys(conn, inBuf, outBuf, t, keyList)
	numKeys := getNumKeys(conn, inBuf, outBuf, t, pkp)

	if numKeys != 2 {
		t.Error(fmt.Sprintf("Returned %d keys instead of 2.", numKeys))
	}
	server.StopServer()
}

func enablePush(conn *transport.Conn, inBuf []byte, outBuf []byte, t *testing.T) {
	true_ := true
	command := &proto.ClientToServer{
		ReceiveEnvelopes: &true_,
	}
	writeProtobuf(conn, outBuf, command, t)
	receiveProtobuf(conn, inBuf, t)
}

func dropMessage(t *testing.T, server *Server, uid *[32]byte, message []byte) {
	oldConn, err := net.Dial("tcp", server.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer oldConn.Close()

	pkp, skp, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	conn, _, err := transport.Handshake(oldConn, pkp, skp, nil, proto.SERVER_MESSAGE_SIZE)
	handleError(err, t)

	inBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)
	outBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)

	uploadMessageToUser(conn, inBuf, outBuf, t, uid, message)
}

func TestPushNotifications(t *testing.T) {
	//t.Skipf("server_test.go:22: read tcp [::1]:55166: i/o timeout; server_test.go:46: Server returned nil status; nil dereference at server_test.go:48")
	dir, err := ioutil.TempDir("", "testdb")
	handleError(err, t)

	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	handleError(err, t)

	defer db.Close()

	server, conn, inBuf, outBuf, pkp := setUpServerTest(db, t)
	defer conn.Close()

	envelope1 := []byte("First")
	envelope2 := []byte("Second")
	envelope3 := []byte("Third")

	createAccount(conn, inBuf, outBuf, t)
	enablePush(conn, inBuf, outBuf, t)
	dropMessage(t, server, pkp, envelope1)
	dropMessage(t, server, pkp, envelope2)
	r1 := receiveProtobuf(conn, inBuf, t)
	r2 := receiveProtobuf(conn, inBuf, t)
	dropMessage(t, server, pkp, envelope3)
	r3 := receiveProtobuf(conn, inBuf, t)

	if !bytes.Equal(r1.Envelope, envelope1) {
		t.Error(fmt.Sprintf("first message mismatch: \"%s\" != \"%s\"", r1.Envelope, envelope1))
	}
	if !bytes.Equal(r2.Envelope, envelope2) {
		t.Error(fmt.Sprintf("first message mismatch: \"%s\" != \"%s\"", r2.Envelope, envelope2))
	}
	if !bytes.Equal(r3.Envelope, envelope3) {
		t.Error(fmt.Sprintf("first message mismatch: \"%s\" != \"%s\"", r3.Envelope, envelope3))
	}

	messageList := []*[32]byte{(*[32]byte)(r1.MessageId), (*[32]byte)(r2.MessageId), (*[32]byte)(r3.MessageId)}

	for _, element := range messageList {
		if element == nil {
			t.Fatal("Message ID nil.")
		}
	}

	deleteMessages(conn, inBuf, outBuf, t, messageList)

	newMessageList := listUserMessages(conn, inBuf, outBuf, t)

	if !(len(newMessageList) == 0) {
		t.Error("Not all messages properly deleted")
	}

	server.StopServer()
}
