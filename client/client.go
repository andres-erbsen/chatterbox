package main

import (
	"code.google.com/p/go.crypto/nacl/box"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"crypto/rand"
	"errors"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/dename/client"
	"net"
)

const MAX_MESSAGE_SIZE = 16 * 1024
const KEYS_TO_GENERATE = 100

type Client struct {
	dename                []byte
	conn                  *transport.Conn
	skAuth                *[32]byte
	serverTransportPubkey *[32]byte
	denameClient          *client.Client
}

func StartClient(dename []byte, addr string, skAuth *[32]byte, config *client.Config) error {
	oldConn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	conn, _, err := transport.Handshake(oldConn, pkp, skp, serverTransportPubkey, MAX_MESSAGE_SIZE)
	if err != nil {
		return err
	}
	denameClient, err := client.NewClient(config, nil, nil)
	if err != nil {
		return err
	}
	client := &Client{
		dename:       dename,
		conn:         conn,
		skAuth:       skAuth,
		denameClient: denameClient,
	}
}

func (client *Client) createAccount() error {
	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	if err := createAccount(client.conn, inBuf, outBuf); err != nil {
		return err
	}
	return nil
}

func (client *Client) generateUploadKeys(pkList [][32]byte) error {
	err := uploadKeys(client.conn, inBuf, outBuf, &pkList)
	if err != nil {
		return err
	}
	return nil
}

func encryptAuthFirst(user *[32]byte, msg []byte, dename []byte, skAuth *[32]byte, config *client.Config) (*ratchet.Ratchet, error) {

	denameClient, err := client.NewClient(config, nil, nil)
	if err != nil {
		return nil, err
	}

	ratch := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(skAuth),
		CheckAuth: CheckAuthWith(denameClient),
		Rand:      nil,
		Now:       nil,
	}

	msg, err := protobuf.Marshal(&proto.Message{
		Subject:  nil,
		Contents: msg,
		Dename:   dename,
	})
	if err != nil {
		return nil, err
	}

	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	//dename lookup, see what server is
	userKey, err := getKey(client.conn, inBuf, outBuf, user)
	if err != nil {
		return nil, err
	}

	out := append([]byte{}, userKey[:]...)
	out = ratch.EncryptFirst(out, msg, pkb0)

	inBuf = make([]byte, MAX_MESSAGE_SIZE)
	outBuf = make([]byte, MAX_MESSAGE_SIZE)

	if err := uploadMessageToUser(client.conn, inBuf, outBuf, user, out); err != nil {
		return nil, err
	}
	return ratch, nil
}

func (client *Client) decryptAuthFirst(in []byte, skList [][32]byte) (*ratchet.Ratchet, []byte, int, error) {
	ratch := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(client.skAuth),
		CheckAuth: CheckAuthWith(client.denameClient),
		Rand:      nil,
		Now:       nil,
	}

	for i, key := range skList {
		msg, err := ratch.DecryptFirst(in[32:], key)
		if err == nil {
			return msg, i, nil
		}
	}
	return nil, nil, errors.New("Invalid message received.") //TODO: Should I make the error message something different?
}

//func (client *Client) encryptAuth(user *[32]byte, msg []byte, ratch *ratchet.Ratchet) error {

//msg, err := protobuf.Marshal(&proto.Message{
//Subject:  nil,
//Contents: msg,
//Dename:   client.dename,
//})
//if err != nil {
//return nil, err
//}

//inBuf := make([]byte, MAX_MESSAGE_SIZE)
//outBuf := make([]byte, MAX_MESSAGE_SIZE)

//theirAuthPublic := (*[32]byte)(ratch.GetTheirAuthPublic)
//out := append([]byte{}, theirAuthPublic()[:]...)
//out = ratch.EncryptFirst(out, msg)

//inBuf = make([]byte, MAX_MESSAGE_SIZE)
//outBuf = make([]byte, MAX_MESSAGE_SIZE)

//if err := uploadMessageToUser(client.conn, inBuf, outBuf, user, out); err != nil {
//return nil, err
//}
//return ratch, nil
//}
