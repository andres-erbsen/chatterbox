package main

/*
import (
	"code.google.com/p/go.crypto/nacl/box"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"crypto/rand"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/dename/client"
	"net"
)

const MAX_MESSAGE_SIZE = 16 * 1024
const KEYS_TO_GENERATE = 100

type Client struct {
	dename       []byte
	conn         *transport.Conn
	skAuth       *[32]byte
	serverTransportPubkey       *[32]byte
	denameClient *client.Client
	firstKeys [][32]byte
}

func StartClient(dename []byte, addr string, serverTransportPubkey, skAuth *[32]byte, config *client.Config, firstKeys [][32]byte) error {
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
		firstKeys: firstKeys,
	}
	//go client.RunClient()
}

func (client *Client) RunClient() error {
	//TODO: Start daemon?
}

func (client *Client) createAccount() error {
	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	if err := createAccount(client.conn, inBuf, outBuf); err != nil {
		return err
	}
	return nil
}

func (client *Client) generateUploadKeys() ([][32]byte, error) {
	skList := make([][32]byte, 0, 64)
	pkList := make([][32]byte, 0, 64)
	for _, _ := range KEYS_TO_GENERATE {
		pk, sk, err := box.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		skList = append(skList, *sk)
		pkList = append(pkList, *pk)
	}

	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	err := uploadKeys(client.conn, inBuf, outBuf, &pkList)
	if err != nil {
		return nil, err
	}
	for _, sk := range skList {
		append(client.firstKeys, sk)
	}
	return skList, nil
}

func (client *Client) encryptAuthFirst(user *[32]byte, msg []byte) (*ratchet.Ratchet, error) {
	ratch := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(client.skAuth),
		CheckAuth: CheckAuthWith(client.denameClient),
		Rand:      nil,
		Now:       nil,
	}

	msg, err := protobuf.Marshal(&proto.Message{
		Subject:  nil,
		Contents: msg,
		Dename:   client.dename,
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

func (client *Client) decryptAuthFirst(in []byte) (msg []byte, error) {
	ratch := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(client.skAuth),
		CheckAuth: CheckAuthWith(client.denameClient),
		Rand:      nil,
		Now:       nil,
	}

	//TODO:, ugh, how do we check for reasonableness
	//msg, err:= ratch.DecryptFirst(in[32:],)
}

func (client *Client) encryptAuth(user *[32]byte, msg []byte, ratch *ratchet.Ratchet) error {

	msg, err := protobuf.Marshal(&proto.Message{
		Subject:  nil,
		Contents: msg,
		Dename:   client.dename,
	})
	if err != nil {
		return nil, err
	}

	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	theirAuthPublic := (*[32]byte)(ratch.GetTheirAuthPublic)
	out := append([]byte{}, theirAuthPublic()[:]...)
	out = ratch.EncryptFirst(out, msg)

	inBuf = make([]byte, MAX_MESSAGE_SIZE)
	outBuf = make([]byte, MAX_MESSAGE_SIZE)

	if err := uploadMessageToUser(client.conn, inBuf, outBuf, user, out); err != nil {
		return nil, err
	}
	return ratch, nil
}
*/
