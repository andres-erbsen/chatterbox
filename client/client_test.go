package client

import (
	"bytes"
	"code.google.com/p/go.crypto/nacl/box"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"crypto/rand"
	"fmt"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/dename/client"
	testutil2 "github.com/andres-erbsen/dename/server/testutil" //TODO: Move MakeToken to TestUtil
	"github.com/andres-erbsen/dename/testutil"
	"testing"
	"time"
)

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
		Subject:  "",
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

func createNewUser(name []byte, t *testing.T, config *client.Config) (*[32]byte, *client.Client) {
	newClient, err := client.NewClient(config, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	//TODO: All these names are horrible, please change them
	pkAuth, skAuth, err := box.GenerateKey(rand.Reader)

	chatProfile := &proto.Profile{
		ServerAddressTCP:  "",
		ServerPortTCP:     -1,
		ServerTransportPK: (proto.Byte32)([32]byte{}),
		UserIDAtServer:    (proto.Byte32)([32]byte{}),
		KeySigningKey:     (proto.Byte32)([32]byte{}),
		MessageAuthKey:    (proto.Byte32)(*pkAuth),
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
		t.Error("Correct profile not added to server.")
		fmt.Printf("profile: %v\n", profile)
		fmt.Printf("profile2: %v\n", profile2)
	}

	return skAuth, newClient
}
