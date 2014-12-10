package client

import (
	"code.google.com/p/go.crypto/curve25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	//"github.com/agl/ed25519"
	"github.com/andres-erbsen/dename/client"
	//"github.com/andres-erbsen/dename/protocol"
	"bytes"
	"code.google.com/p/go.crypto/nacl/box"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"crypto/subtle"
	"errors"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	testutil2 "github.com/andres-erbsen/dename/server/testutil" //TODO: Move MakeToken to TestUtil
	"github.com/andres-erbsen/dename/testutil"
	//"io"
	"testing"
	"time"
)

const PROFILE_FIELD_ID = 1984

func TestAliceTalksToBob(t *testing.T) {
	dir, err := ioutil.TempDir("", "testdb")
	handleError(err, t)

	defer os.RemoveAll(dir)
	db, err := leveldb.OpenFile(dir, nil)
	handleError(err, t)

	defer db.Close()

	server, pks := setUpServerTest(db, t)

	connA, inBufA, outBufA, pkpA := setUpClientTest(server, t)
	connB, inBufB, outBufB, pkpB := setUpClientTest(server, t)

	config, f := testutil.SingleServer(t)
	defer f()
	time.Sleep(100)

	//func createNewUser(name []byte, t *testing.T, config *client.Config, serverAddr string, pkTransport *[32]byte, idServer *[32]byte, signingKey *[32]byte, authKey *[32]byte) (*[32]byte, *client.Client) {
	pkTransportA, skTransportA, err := box.GenerateKey(rand.Reader)
	handleError(err)

	pkTransportB, skTransportA, err := box.GenerateKey(rand.Reader)
	handleError(err)

	ska, dnmca := createNewUser([]byte("Alice"), t, config, server.listener.Addr().String(), pkTransportA, pks)
	skb, dnmcb := createNewUser([]byte("Bob"), t, config)

	profileA
	profileB

	msg = []byte{"Envelope1"}
	ratch, err := encryptAuthFirst(msg, []byte("Bob"), ska, config)
	handleError(err)

	clientA := StartClient([]byte("Alice"))
	clientB := StartClient([]byte("Bob"), server.listener.Addr().String(), skb, connB, config)

	//clientB.decryptAuthFirst(

	//clientB := StartClient([]byte("Bob),
}

func setUpClientTest(server *Server, t *testing.T) (*transport.Conn, []byte, []byte, *[32]byte) {
	oldConn, err := net.Dial("tcp", server.listener.Addr().String())
	handleError(err, t)

	pkp, skp, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	conn, _, err := transport.Handshake(oldConn, pkp, skp, nil, MAX_MESSAGE_SIZE)
	handleError(err, t)

	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	return conn, inBuf, outBuf, pkp
}

func setUpServerTest(db *leveldb.DB, t *testing.T) (*Server, *[32]byte) {
	shutdown := make(chan struct{})

	pks, sks, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	server, err := StartServer(db, shutdown, pks, sks)
	handleError(err, t)

	return server, nil
}

func handleError(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
	}
}

func TestMessageEncryptionAuthentication(t *testing.T) {
	config, f := testutil.SingleServer(t)
	defer f()
	time.Sleep(100)

	ska, dnmca := createNewUser([]byte("Alice"), t, config)
	skb, dnmcb := createNewUser([]byte("Bob"), t, config)

	ratchA := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(ska),
		CheckAuth: CheckAuthWith(dnmca),
		Rand:      nil,
		Now:       nil,
	}
	ratchB := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(skb),
		CheckAuth: CheckAuthWith(dnmcb),
		Rand:      nil,
		Now:       nil,
	}

	//pka0, ska0, err := box.GenerateKey(rand.Reader)
	//handleError(err, t)
	pkb0, skb0, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := protobuf.Marshal(&proto.Message{
		Subject:  nil,
		Contents: []byte("Message"),
		Dename:   []byte("Alice"),
	})
	if err != nil {
		t.Fatal(err)
	}
	out := append([]byte{}, (*pkb0)[:]...)

	out = ratchA.EncryptFirst(out, msg, pkb0)
	msg2, err := ratchB.DecryptFirst(out[32:], skb0)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(msg, msg2) {
		t.Error("Original and decrypted message not the same.")
	}
}

func createNewUser(name []byte, t *testing.T, config *client.Config, serverAddr string, pkTransport *[32]byte, idServer *[32]byte, signingKey *[32]byte, authKey *[32]byte) (*[32]byte, *client.Client) {
	newClient, err := client.NewClient(config, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	//TODO: All these names are horrible, please change them
	pkAuth, skAuth, err := box.GenerateKey(rand.Reader)

	chatProfile := &proto.Profile{
		ServerAddressTCP:  []byte(serverAddr),
		ServerTransportPK: (proto.Byte32)(pkTransport),
		UserIDAtServer:    (proto.Byte32)(idServer),
		KeySigningKey:     (proto.Byte32)(signingKey),
		MessageAuthKey:    (proto.Byte32)(authKey),
	}

	chatProfileBytes, err := protobuf.Marshal(chatProfile)
	if err != nil {
		t.Fatal(err)
	}

	profile, sk, err := client.NewProfile(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	client.SetProfileField(profile, PROFILE_FIELD_ID, chatProfileBytes)

	err = newClient.Register(sk, name, profile, testutil2.MakeToken())
	if err != nil {
		t.Fatal(err)
	}

	//Remove this outside of the test
	profile2, err := newClient.Lookup(name)
	if !profile.Equal(profile2) {
		t.Errorf("Correct profile not added to server:\nprofile: %v\nprofile2 %v\n", profile, profile2)
	}

	return skAuth, newClient
}

//func FillAuthWith(ourAuthPrivate *[32]byte) func([]byte, []byte, *[32]byte) {
//return func(tag, data []byte, theirAuthPublic *[32]byte) {
//var sharedAuthKey [32]byte
//curve25519.ScalarMult(&sharedAuthKey, ourAuthPrivate, theirAuthPublic)
//h := hmac.New(sha256.New, sharedAuthKey[:])
//h.Write(data)
//h.Sum(nil)
//copy(tag, h.Sum(nil))
//}
//}

//func CheckAuthWith(dnmc *client.Client) func([]byte, []byte, []byte, *[32]byte) error {
//return func(tag, data, msg []byte, ourAuthPrivate *[32]byte) error {
//var sharedAuthKey [32]byte
//message := new(proto.Message)
//if err := message.Unmarshal(msg); err != nil {
//return err
//}
//profile, err := dnmc.Lookup(message.Dename)
//if err != nil {
//return err
//}

//chatProfileBytes, err := client.GetProfileField(profile, PROFILE_FIELD_ID)
//if err != nil {
//return err
//}

//chatProfile := new(proto.Profile)
//if err := chatProfile.Unmarshal(chatProfileBytes); err != nil {
//return err
//}

//theirAuthPublic := (*[32]byte)(&chatProfile.MessageAuthKey)

//curve25519.ScalarMult(&sharedAuthKey, ourAuthPrivate, theirAuthPublic)
//h := hmac.New(sha256.New, sharedAuthKey[:])
//h.Write(data)
//if subtle.ConstantTimeCompare(tag, h.Sum(nil)[:len(tag)]) == 0 {

//return errors.New("Authentication failed: failed to reproduce envelope auth tag using the current auth pubkey from dename")
//}
//return nil
//}
//}
