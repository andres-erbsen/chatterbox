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

const MAX_MESSAGE_SIZE = 16 * 1024
const PROFILE_FIELD_ID = 1984

func ToProtoByte32List(list [][32]byte) []proto.Byte32 {
	newList := make([]proto.Byte32, 0)
	for _, element := range list {
		newList = append(newList, (proto.Byte32)(element))
	}
	return newList
}

func To32ByteList(list []proto.Byte32) [][32]byte {
	newList := make([][32]byte, 0, 0)
	for _, element := range list {
		newList = append(newList, ([32]byte)(element))
	}
	return newList
}

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

	return To32ByteList(response.MessageList), nil
}

func DownloadEnvelope(conn *transport.Conn, connToServer *ConnectionToServer, outBuf []byte, messageHash *[32]byte) error {
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
		signedKey := ed25519.Sign(sk, key[:])
		pkList = append(pkList, append(append([]byte{}, key[:]...), signedKey[:]...))
	}
	return pkList
}

func CreateServerConn(dename []byte, denameClient *client.Client) (*transport.Conn, error) {
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
	pkTransport := ([32]byte)(chatProfile.ServerTransportPK)

	oldConn, err := net.Dial("tcp", addr+":1984")
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

	return conn, nil
}

func EncryptAuthFirst(dename []byte, msg []byte, skAuth *[32]byte, userKey *[32]byte, denameClient *client.Client) ([]byte, *ratchet.Ratchet, error) {
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
		return nil, nil, err
	}

	out := append([]byte{}, (*userKey)[:]...)
	out = ratch.EncryptFirst(out, message, userKey)

	return out, ratch, nil
}

func EncryptAuth(dename []byte, msg []byte, ratch *ratchet.Ratchet) ([]byte, *ratchet.Ratchet, error) {
	message, err := protobuf.Marshal(&proto.Message{
		Subject:  nil,
		Contents: msg,
		Dename:   dename,
	})
	if err != nil {
		return nil, nil, err
	}

	out := []byte{}
	out = ratch.Encrypt(out, message)

	return out, ratch, nil
}

func DecryptAuthFirst(in []byte, skList []*[32]byte, skAuth *[32]byte, denameClient *client.Client) (*ratchet.Ratchet, []byte, int, error) {
	ratch := &ratchet.Ratchet{
		FillAuth:  FillAuthWith(skAuth),
		CheckAuth: CheckAuthWith(denameClient),
	}

	for i, key := range skList {
		msg, err := ratch.DecryptFirst(in[32:], key)
		if err == nil {
			return ratch, msg, i, nil
		}
	}
	return nil, nil, -1, errors.New("Invalid message received.") //TODO: Should I make the error message something different?
}

func DecryptAuth(in []byte, ratch *ratchet.Ratchet) (*ratchet.Ratchet, []byte, error) {
	msg, err := ratch.Decrypt(in[32:])
	if err != nil {
		return nil, nil, errors.New("Invalid message.") //TODO: Should I make the error message something different?
	}
	return ratch, msg, nil
}

func DeleteMessages(conn *transport.Conn, connToServer *ConnectionToServer, outBuf []byte, messageList [][32]byte) error {
	deleteMessages := &proto.ClientToServer{
		DeleteMessages: ToProtoByte32List(messageList),
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

func GetKey(conn *transport.Conn, inBuf []byte, outBuf []byte, pk *[32]byte, dename []byte, denameClient *client.Client) (*[32]byte, error) {
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

	pkSig := (*[32]byte)(&chatProfile.KeySigningKey)

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
	copy(userKey[:], response.SignedKey[32:(32+64)])

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
	size, err := message.MarshalTo(outBuf)
	if err != nil {
		return err
	}
	conn.WriteFrame(outBuf[:size])
	return nil
}

func ReceiveProtobuf(conn *transport.Conn, inBuf []byte) (*proto.ServerToClient, error) {
	response := new(proto.ServerToClient)
	conn.SetDeadline(time.Now().Add(time.Second))
	num, err := conn.ReadFrame(inBuf)
	if err != nil {
		return nil, err
	}
	if err := response.Unmarshal(inBuf[:num]); err != nil {
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

func CreateTestDenameAccount(name []byte, denameClient *client.Client, secretConfig *proto.LocalAccountConfig, serverAddr string, serverPk *[32]byte, t *testing.T) {
	//TODO: move this to test?
	//TODO: All these names are horrible, please change them
	chatProfile := &proto.Profile{
		ServerAddressTCP:  serverAddr,
		ServerTransportPK: (proto.Byte32)(*serverPk),
	}

	err := GenerateLongTermKeys(secretConfig, chatProfile, rand.Reader)

	if err != nil {
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
		if err := message.Unmarshal(msg); err != nil {
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
		h.Write(data)
		if subtle.ConstantTimeCompare(tag, h.Sum(nil)[:len(tag)]) == 0 {

			return errors.New("Authentication failed: failed to reproduce envelope auth tag using the current auth pubkey from dename")
		}
		return nil
	}
}
