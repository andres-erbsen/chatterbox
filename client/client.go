package client

import (
	"code.google.com/p/go.crypto/nacl/box"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"crypto/rand"
	"errors"
	"github.com/agl/ed25519"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/chatterbox/transport"
	"github.com/andres-erbsen/dename/client"
	"net"
)

const MAX_MESSAGE_SIZE = 16 * 1024
const KEYS_TO_GENERATE = 100
const PROFILE_FIELD_ID = 1984 //TODO: Sync this constant

type Client struct {
	dename                []byte
	conn                  *transport.Conn
	skAuth                *[32]byte
	serverTransportPubkey *[32]byte
	denameClient          *client.Client
}

func StartClient(dename []byte, addr string, skAuth *[32]byte, skp *[32]byte, pkp *[32]byte, pkTransport *[32]byte, config *client.Config) (*Client, error) {
	oldConn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	conn, _, err := transport.Handshake(oldConn, pkp, skp, pkTransport, MAX_MESSAGE_SIZE)
	if err != nil {
		return nil, err
	}
	denameClient, err := client.NewClient(config, nil, nil)
	if err != nil {
		return nil, err
	}
	newClient := &Client{
		dename:       dename,
		conn:         conn,
		skAuth:       skAuth,
		denameClient: denameClient,
	}
	return newClient, nil
}

func (client *Client) createAccount() error {
	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	if err := CreateAccount(client.conn, inBuf, outBuf); err != nil {
		return err
	}
	return nil
}

func (client *Client) listMessages() (*[][32]byte, error) {
	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	messageList, err := listUserMessages(client.conn, inBuf, outBuf)
	if err != nil {
		return nil, err
	}
	return messageList, nil
}

func (client *Client) downloadMessage(messageHash *[32]byte) ([]byte, error) {
	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	message, err := downloadEnvelope(client.conn, inBuf, outBuf, messageHash)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (client *Client) uploadKeys(keys *[][32]byte, sk *[64]byte) error {
	pkList := make([][]byte, 0)
	for _, key := range *keys {
		signedKey := ed25519.Sign(sk, key[:])
		pkList = append(pkList, append(append([]byte{}, key[:]...), signedKey[:]...))
	}

	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	err := uploadKeys(client.conn, inBuf, outBuf, &pkList)
	if err != nil {
		return err
	}
	return nil
}

func encryptAuthFirst(msg []byte, dename []byte, skAuth *[32]byte, config *client.Config) (*ratchet.Ratchet, error) {

	denameClient, err := client.NewClient(config, nil, nil)
	if err != nil {
		return nil, err
	}

	ratch := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(skAuth),
		CheckAuth: CheckAuthWith(denameClient),
	}

	message, err := protobuf.Marshal(&proto.Message{
		Subject:  nil,
		Contents: msg,
		Dename:   dename,
	})
	if err != nil {
		return nil, err
	}

	profile, err := denameClient.Lookup(dename)
	if err != nil {
		return nil, err
	}

	chatProfileBytes, err := client.GetProfileField(profile, PROFILE_FIELD_ID)
	if err != nil {
		return nil, err
	}

	chatProfile := new(proto.Profile)
	if err := chatProfile.Unmarshal(chatProfileBytes); err != nil {
		return nil, err
	}

	addr := chatProfile.ServerAddressTCP
	user := ([32]byte)(chatProfile.UserIDAtServer)
	pkTransport := ([32]byte)(chatProfile.ServerTransportPK)
	pkSig := ([32]byte)(chatProfile.KeySigningKey)

	oldConn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	pkp, skp, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	conn, _, err := transport.Handshake(oldConn, pkp, skp, &pkTransport, MAX_MESSAGE_SIZE)
	if err != nil {
		return nil, err
	}

	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)
	//dename lookup, see what server is
	keySig, err := getKey(conn, inBuf, outBuf, &user)

	var userKey [32]byte
	copy(userKey[:], keySig[:32])

	var sig [64]byte
	copy(userKey[:], keySig[32:(32+64)])

	if !ed25519.Verify(&pkSig, userKey[:], &sig) {
		return nil, errors.New("Improperly signed key returned")
	}

	//TODO: Put this part that creates the message elsewhere
	out := append([]byte{}, userKey[:]...)
	out = ratch.EncryptFirst(out, message, &userKey)

	inBuf = make([]byte, MAX_MESSAGE_SIZE)
	outBuf = make([]byte, MAX_MESSAGE_SIZE)

	if err := uploadMessageToUser(conn, inBuf, outBuf, &user, out); err != nil {
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
		msg, err := ratch.DecryptFirst(in[32:], &key)
		if err == nil {
			return ratch, msg, i, nil
		}
	}
	return nil, nil, -1, errors.New("Invalid message received.") //TODO: Should I make the error message something different?
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
