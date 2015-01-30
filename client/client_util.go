package client

import (
	"code.google.com/p/go.crypto/curve25519"
	"code.google.com/p/go.crypto/nacl/box"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"github.com/agl/ed25519"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/chatterbox/transport"
	"github.com/andres-erbsen/dename/client"
	testutil2 "github.com/andres-erbsen/dename/server/testutil" //TODO: Move MakeToken to TestUtil
	"io"
	"net"
	"testing"
	"time"
)

const PROFILE_FIELD_ID = 1984
const ENCRYPT_ADDED_LEN = 168
const ENCRYPT_FIRST_ADDED_LEN = 200

func ReceiveReply(connToServer *ConnectionToServer) (*proto.ServerToClient, error) {
	response := <-connToServer.ReadReply //TODO: Timeout
	return response, nil
}

func CreateAccount(conn *transport.Conn, inBuf []byte, outBuf []byte) error {
	command := &proto.ClientToServer{
		CreateAccount: protobuf.Bool(true),
	}
	if err := WriteProtobuf(conn, outBuf, command); err != nil {
		return err
	}

	_, err := ReceiveProtobuf(conn, inBuf)
	if err != nil {
		return err
	}
	return nil
}

func ListUserMessages(conn *transport.Conn, connToServer *ConnectionToServer, outBuf []byte) ([][32]byte, error) {
	listMessages := &proto.ClientToServer{
		ListMessages: protobuf.Bool(true),
	}
	if err := WriteProtobuf(conn, outBuf, listMessages); err != nil {
		return nil, err
	}

	response, err := ReceiveReply(connToServer)
	if err != nil {
		return nil, err
	}

	return proto.To32ByteList(response.MessageList), nil
}

func RequestMessage(conn *transport.Conn, connToServer *ConnectionToServer, outBuf []byte, messageHash *[32]byte) error {
	getEnvelope := &proto.ClientToServer{
		DownloadEnvelope: (*proto.Byte32)(messageHash),
	}
	if err := WriteProtobuf(conn, outBuf, getEnvelope); err != nil {
		return err
	}
	return nil
}

func SignKeys(keys []*[32]byte, sk *[64]byte) [][]byte {

	pkList := make([][]byte, 0)
	for _, key := range keys {
		signature := ed25519.Sign(sk, key[:])
		pkList = append(pkList, append(append([]byte{}, key[:]...), signature[:]...))
	}
	return pkList
}

func CreateTestAccount(name string, denameClient *client.Client, secretConfig *proto.LocalAccountConfig, serverAddr string, serverPk *[32]byte, t *testing.T) *transport.Conn {

	CreateTestDenameAccount(name, denameClient, secretConfig, serverAddr, serverPk, t)
	conn := CreateTestHomeServerConn(name, denameClient, secretConfig, t)

	inBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)
	outBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)

	err := CreateAccount(conn, inBuf, outBuf)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func CreateTestHomeServerConn(dename string, denameClient *client.Client, secretConfig *proto.LocalAccountConfig, t *testing.T) *transport.Conn {
	profile, err := denameClient.Lookup(dename)
	if err != nil {
		t.Fatal(err)
	}

	chatProfileBytes, err := client.GetProfileField(profile, PROFILE_FIELD_ID)
	if err != nil {
		t.Fatal(err)
	}

	chatProfile := new(proto.Profile)
	if err := chatProfile.Unmarshal(chatProfileBytes); err != nil {
		t.Fatal(err)
	}

	addr := chatProfile.ServerAddressTCP
	port := chatProfile.ServerPortTCP
	pkTransport := ([32]byte)(chatProfile.ServerTransportPK)
	pkp := (*[32]byte)(&chatProfile.UserIDAtServer)

	oldConn, err := net.Dial("tcp", net.JoinHostPort(addr, fmt.Sprint(port)))
	if err != nil {
		t.Fatal(err)
	}

	skp := (*[32]byte)(&secretConfig.TransportSecretKeyForServer)

	conn, _, err := transport.Handshake(oldConn, pkp, skp, &pkTransport, proto.SERVER_MESSAGE_SIZE)
	if err != nil {
		t.Fatal(err)
	}

	return conn
}

func CreateHomeServerConn(addr string, pkp, skp, pkTransport *[32]byte) (*transport.Conn, error) {
	oldConn, err := net.Dial("tcp", net.JoinHostPort(addr, "1984"))
	if err != nil {
		return nil, err
	}

	conn, _, err := transport.Handshake(oldConn, pkp, skp, pkTransport, proto.SERVER_MESSAGE_SIZE)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func CreateForeignServerConn(dename string, denameClient *client.Client, addr string, port int, pkTransport *[32]byte) (*transport.Conn, error) {

	oldConn, err := net.Dial("tcp", net.JoinHostPort(addr, fmt.Sprint(port)))
	if err != nil {
		return nil, err
	}

	pkp, skp, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	conn, _, err := transport.Handshake(oldConn, pkp, skp, pkTransport, proto.SERVER_MESSAGE_SIZE)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func EncryptAuthFirst(message []byte, skAuth *[32]byte, userKey *[32]byte, denameClient *client.Client) ([]byte, *ratchet.Ratchet, error) {
	ratch := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(skAuth),
		CheckAuth: CheckAuthWith(denameClient),
	}

	out := append([]byte{}, (*userKey)[:]...)
	paddedMsg := proto.Pad(message, proto.MAX_MESSAGE_SIZE-ENCRYPT_FIRST_ADDED_LEN-len(out))
	out = ratch.EncryptFirst(out, paddedMsg, userKey)

	return out, ratch, nil
}

func EncryptAuth(message []byte, ratch *ratchet.Ratchet) ([]byte, *ratchet.Ratchet, error) {
	paddedMsg := proto.Pad(message, proto.MAX_MESSAGE_SIZE-ENCRYPT_ADDED_LEN)
	out := ratch.Encrypt(nil, paddedMsg)

	return out, ratch, nil
}

func DecryptAuthFirst(in []byte, pkList []*[32]byte, skList []*[32]byte, skAuth *[32]byte, denameClient *client.Client) (*ratchet.Ratchet, []byte, int, error) {
	ratch := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(skAuth),
		CheckAuth: CheckAuthWith(denameClient),
	}

	if len(in) < 32 {
		return nil, nil, -1, errors.New("Message length incorrect.")
	}
	var pkAuth [32]byte
	copy(pkAuth[:], in[:32])

	envelope := in[32:]
	for i, pk := range pkList {
		if *pk == pkAuth {
			msg, err := ratch.DecryptFirst(envelope, skList[i])
			if err == nil {
				unpadMsg := proto.Unpad(msg)
				return ratch, unpadMsg, i, nil
			}
		}
	}
	return nil, nil, -1, errors.New("Invalid first message received.") //TODO: Should I make the error message something different?
}
func DecryptAuth(in []byte, ratch *ratchet.Ratchet) (*ratchet.Ratchet, []byte, error) {
	msg, err := ratch.Decrypt(in)
	if err != nil {
		return nil, nil, err
	}
	unpadMsg := proto.Unpad(msg)
	return ratch, unpadMsg, nil
}

func DeleteMessages(conn *transport.Conn, connToServer *ConnectionToServer, outBuf []byte, messageList [][32]byte) error {
	deleteMessages := &proto.ClientToServer{
		DeleteMessages: proto.ToProtoByte32List(messageList),
	}
	if err := WriteProtobuf(conn, outBuf, deleteMessages); err != nil {
		return err
	}

	_, err := ReceiveReply(connToServer)
	if err != nil {
		return err
	}
	return nil
}

func UploadKeys(conn *transport.Conn, connToServer *ConnectionToServer, outBuf []byte, keyList [][]byte) error {
	uploadKeys := &proto.ClientToServer{
		UploadSignedKeys: keyList,
	}
	if err := WriteProtobuf(conn, outBuf, uploadKeys); err != nil {
		return nil
	}

	_, err := ReceiveReply(connToServer)
	if err != nil {
		return err
	}
	return nil
}

func GetKey(conn *transport.Conn, inBuf []byte, outBuf []byte, pk *[32]byte, dename string, pkSig *[32]byte) (*[32]byte, error) {
	getKey := &proto.ClientToServer{
		GetSignedKey: (*proto.Byte32)(pk),
	}
	if err := WriteProtobuf(conn, outBuf, getKey); err != nil {
		return nil, err
	}

	response, err := ReceiveProtobuf(conn, inBuf)
	if err != nil {
		return nil, err
	}

	var userKey [32]byte
	copy(userKey[:], response.SignedKey[:32])

	var sig [64]byte
	copy(sig[:], response.SignedKey[32:(32+64)])

	if !ed25519.Verify(pkSig, userKey[:], &sig) {
		return nil, errors.New("Improperly signed key returned")
	}

	return &userKey, nil
}

func GetNumKeys(conn *transport.Conn, connToServer *ConnectionToServer, outBuf []byte) (int64, error) {
	getNumKeys := &proto.ClientToServer{
		GetNumKeys: protobuf.Bool(true),
	}
	if err := WriteProtobuf(conn, outBuf, getNumKeys); err != nil {
		return 0, err
	}

	response, err := ReceiveReply(connToServer)
	if err != nil {
		return 0, err
	}
	return *response.NumKeys, nil
}

func EnablePush(conn *transport.Conn, connToServer *ConnectionToServer, outBuf []byte) error {
	true_ := true
	command := &proto.ClientToServer{
		ReceiveEnvelopes: &true_,
	}
	if err := WriteProtobuf(conn, outBuf, command); err != nil {
		return err
	}
	_, err := ReceiveReply(connToServer)
	if err != nil {
		return err
	}
	return nil
}

func UploadMessageToUser(conn *transport.Conn, inBuf []byte, outBuf []byte, pk *[32]byte, envelope []byte) error {
	message := &proto.ClientToServer_DeliverEnvelope{
		User:     (*proto.Byte32)(pk),
		Envelope: envelope,
	}
	deliverCommand := &proto.ClientToServer{
		DeliverEnvelope: message,
	}
	if err := WriteProtobuf(conn, outBuf, deliverCommand); err != nil {
		return err
	}

	_, err := ReceiveProtobuf(conn, inBuf)
	if err != nil {
		return err
	}
	return nil
}

func WriteProtobuf(conn *transport.Conn, outBuf []byte, message *proto.ClientToServer) error {
	unpadMsg, err := protobuf.Marshal(message)
	if err != nil {
		return err
	}
	padMsg := proto.Pad(unpadMsg, proto.SERVER_MESSAGE_SIZE)
	copy(outBuf, padMsg)

	conn.WriteFrame(outBuf[:proto.SERVER_MESSAGE_SIZE])
	return nil
}

func ReceiveProtobuf(conn *transport.Conn, inBuf []byte) (*proto.ServerToClient, error) {
	response := new(proto.ServerToClient)
	conn.SetDeadline(time.Now().Add(time.Hour))
	num, err := conn.ReadFrame(inBuf)
	if err != nil {
		return nil, err
	}
	unpadMsg := proto.Unpad(inBuf[:num])
	if err := response.Unmarshal(unpadMsg); err != nil {
		return nil, err
	}
	if response.Status == nil {
		return nil, errors.New("Server returned nil status.")
	}
	if *response.Status == proto.ServerToClient_PARSE_ERROR {
		return nil, errors.New("Server threw a parse error.")
	}
	return response, nil
}

func CreateTestDenameAccount(name string, denameClient *client.Client, secretConfig *proto.LocalAccountConfig, serverAddr string, serverPk *[32]byte, t *testing.T) {
	//TODO: move this to test?
	//TODO: All these names are horrible, please change them
	addr, portStr, err := net.SplitHostPort(serverAddr)
	if err != nil {
		t.Fatal(err)
	}
	var port int32
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		t.Fatal(err)
	}

	chatProfile := &proto.Profile{
		ServerAddressTCP:  addr,
		ServerPortTCP:     port,
		ServerTransportPK: (proto.Byte32)(*serverPk),
	}

	if err := GenerateLongTermKeys(secretConfig, chatProfile, rand.Reader); err != nil {
		t.Fatal(err)
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

	err = denameClient.Register(sk, name, profile, testutil2.MakeToken())
	if err != nil {
		t.Fatal(err)
	}
}

func GenerateLongTermKeys(secretConfig *proto.LocalAccountConfig, publicProfile *proto.Profile, rand io.Reader) error {
	if pk, sk, err := box.GenerateKey(rand); err != nil {
		return err
	} else {
		secretConfig.TransportSecretKeyForServer = (proto.Byte32)(*sk)
		publicProfile.UserIDAtServer = (proto.Byte32)(*pk)
	}
	if pk, sk, err := box.GenerateKey(rand); err != nil {
		return err
	} else {
		secretConfig.MessageAuthSecretKey = (proto.Byte32)(*sk)
		publicProfile.MessageAuthKey = (proto.Byte32)(*pk)
	}

	if pk, sk, err := ed25519.GenerateKey(rand); err != nil {
		return err
	} else {
		secretConfig.KeySigningSecretKey = sk[:]
		publicProfile.KeySigningKey = (proto.Byte32)(*pk)
	}
	return nil
}

func FillAuthWith(ourAuthPrivate *[32]byte) func([]byte, []byte, *[32]byte) {
	return func(tag, data []byte, theirAuthPublic *[32]byte) {
		var sharedAuthKey [32]byte
		curve25519.ScalarMult(&sharedAuthKey, ourAuthPrivate, theirAuthPublic)

		var ourAuthPublic [32]byte
		curve25519.ScalarBaseMult(&ourAuthPublic, ourAuthPrivate)

		h := hmac.New(sha256.New, sharedAuthKey[:])
		h.Write(data)
		h.Sum(nil)
		copy(tag, h.Sum(nil))
	}
}

func CheckAuthWith(dnmc *client.Client) func([]byte, []byte, []byte, *[32]byte) error {
	return func(tag, data, msg []byte, ourAuthPrivate *[32]byte) error {
		var sharedAuthKey [32]byte
		message := new(proto.Message)
		unpadMsg := proto.Unpad(msg)
		if err := message.Unmarshal(unpadMsg); err != nil {
			return err
		}

		profile, err := dnmc.Lookup(message.Dename)
		if err != nil {
			return err
		}

		chatProfileBytes, err := client.GetProfileField(profile, PROFILE_FIELD_ID)
		if err != nil {
			return err
		}

		chatProfile := new(proto.Profile)
		if err := chatProfile.Unmarshal(chatProfileBytes); err != nil {
			return err
		}

		theirAuthPublic := (*[32]byte)(&chatProfile.MessageAuthKey)

		curve25519.ScalarMult(&sharedAuthKey, ourAuthPrivate, theirAuthPublic)
		h := hmac.New(sha256.New, sharedAuthKey[:])

		//var bobAuthPublic [32]byte
		//curve25519.ScalarBaseMult(&bobAuthPublic, ourAuthPrivate)

		h.Write(data)
		if subtle.ConstantTimeCompare(tag, h.Sum(nil)[:len(tag)]) == 0 {

			return errors.New("Authentication failed: failed to reproduce envelope auth tag using the current auth pubkey from dename")
		}
		return nil
	}
}
