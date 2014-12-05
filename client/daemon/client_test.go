package main

import (
	"crypto/rand"
	//"github.com/agl/ed25519"
	"github.com/andres-erbsen/dename/client"
	//"github.com/andres-erbsen/dename/protocol"
	"code.google.com/p/go.crypto/nacl/box"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	testutil2 "github.com/andres-erbsen/dename/server/testutil" //TODO: Move MakeToken to TestUtil
	"github.com/andres-erbsen/dename/testutil"
	//"reflect"
	"io"
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

//func handleError(err error, t *testing.T) {
//if err != nil {
//t.Error(err)
//}
//}

func TestMessageEncryptionAuthentication(t *testing.T) {
	config, f := testutil.SingleServer(t)
	defer f()

	createNewUser([]byte("Alice"), t, config)
}

func createNewUser(name []byte, t *testing.T, config *client.Config) {
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

	//TODO: All these names are horrible, please change them
	pkAuth, _, err := box.GenerateKey(rand.Reader)

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
}

func sendFirstMessage(msg []byte) {

	ratch := &ratchet.Ratchet{
		FillAuth:  fillAuth,
		CheckAuth: checkAuth,
		Rand:      nil,
		Now:       nil,
	}
}
