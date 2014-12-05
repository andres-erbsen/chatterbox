package main

import (
	"crypto/rand"
	//"github.com/agl/ed25519"
	"github.com/andres-erbsen/dename/client"
	//"github.com/andres-erbsen/dename/protocol"
	"code.google.com/p/go.crypto/nacl/box"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"github.com/andres-erbsen/chatterbox/proto"
	//"github.com/andres-erbsen/chatterbox/ratchet"
	testutil2 "github.com/andres-erbsen/dename/server/testutil" //TODO: Move MakeToken to TestUtil
	"github.com/andres-erbsen/dename/testutil"
	"reflect"
	//"bytes"
	//"errors"
	//"io"
	"testing"
	"time"
)

const authField = 1984

//func setUpServerTest(db *leveldb.DB, t *testing.T) (*Server, *transport.Conn, []byte, []byte, *[32]byte) {
//shutdown := make(chan struct{})

//pks, sks, err := box.GenerateKey(rand.Reader)
//handleError(err, t)

//server, err := StartServer(db, shutdown, pks, sks)
//handleError(err, t)

//oldConn, err := net.Dial("tcp", server.listener.Addr().String())
//handleError(err, t)

//pkp, skp, err := box.GenerateKey(rand.Reader)
//handleError(err, t)

//conn, _, err := transport.Handshake(oldConn, pkp, skp, nil, MAX_MESSAGE_SIZE)
//handleError(err, t)

//inBuf := make([]byte, MAX_MESSAGE_SIZE)
//outBuf := make([]byte, MAX_MESSAGE_SIZE)

//return server, conn, inBuf, outBuf, pkp
//}

func handleError(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
	}
}

func TestMessageEncryptionAuthentication(t *testing.T) {
	config, f := testutil.SingleServer(t)
	defer f()

	ska = createNewUser([]byte("Alice"), t, config)
	skb = createNewUser([]byte("Bob"), t, config)

	ratchA := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(ska),
		CheckAuth: CheckAuth(),
		Rand:      nil,
		Now:       nil,
	}
	ratchB := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(skb),
		CheckAuth: CheckAuth(),
		Rand:      nil,
		Now:       nil,
	}

	pka0, ska0, err := box.GenerateKey(rand.Reader)
	handleError(err, t)
	pkb0, skb0, err := box.GenerateKey(rand.Reader)
	handleError(err, t)

	msg := []byte("Message")
	out := append([]byte{}, (*pkb0)[:]...)

	out = ratchA.EncryptFirst(outA, msg, pkb0)
	msg2 := ratchB.DecryptFirst(out, skb0)

	if !bytes.Equal(msg, msg2) {
		t.Error("Original and decrypted message not the same.")
	}
}

func createNewUser(name []byte, t *testing.T, config *client.Config) *[32]byte {
	newClient, err := client.NewClient(config, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	profile, sk, err := client.NewProfile(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	client.SetProfileField(profile, 2, sk[:])

	time.Sleep(1)

	err = newClient.Register(sk, name, profile, testutil2.MakeToken())
	if err != nil {
		t.Fatal(err)
	}

	//Remove this outside of the test
	profile2, err := newClient.Lookup(name)
	if !reflect.DeepEqual(profile, profile2) {
		t.Error("Correct profile not added to server.")
	}
	//TODO: All these names are horrible, please change them
	pkAuth, skAuth, err := box.GenerateKey(rand.Reader)

	protoAuth := &proto.Profile{
		ServerAddressTCP:  "",
		ServerTransportPK: (proto.Byte32)([32]byte{}),
		UserIDAtServer:    (proto.Byte32)([32]byte{}),
		KeySigningKey:     (proto.Byte32)([32]byte{}),
		MessageAuthKey:    (proto.Byte32)(*pkAuth),
	}

	auth, err := protobuf.Marshal(protoAuth)
	if err != nil {
		t.Fatal(err)
	}
	client.SetProfileField(profile, authField, auth)

	return skAuth
}

func FillAuthWith(ourAuthPrivate *[32]byte) func([]byte, []byte, [32]byte) {
	return func(tag, data []byte, theirAuthPublic *[32]byte) {
		var sharedAuthKey [32]byte
		curve25519.ScalarMult(&sharedAuthKey, ourAuthPrivate, theirAuthPublic)
		h := hmac.New(sha256.New, sharedKey[:])
		h.Write(data)
		h.Sum(nil)
		copy(tag, h.Sum(nil))
	}
}

func CheckAuth(tag, data, msg []byte, ourAuthPrivate *[32]byte) error {
	var sharedAuthKey [32]byte
	message := new(proto.Message)
	if err := message.Unmarshal(msg); err != nil {
		return err
	}
	profile := message.DenameProfile
	//TODO: Parse this
	curve25519.ScalarMult(&sharedAuthKey, ourAuthPrivate, theirAuthPublic)
	h := hmac.New(sha256.New, sharedKey[:])
	h.Write(data)
	if subtle.ConstantTimeEq(tag, h.Sum(nil)) == 0 {
		return errors.New("Authentication failed.")
	}
	return nil
}
