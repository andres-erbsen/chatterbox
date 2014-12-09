package main

import (
	"code.google.com/p/go.crypto/curve25519"
	"code.google.com/p/go.crypto/nacl/box"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/transport"
	"github.com/andres-erbsen/dename/client"
	testutil2 "github.com/andres-erbsen/dename/server/testutil" //TODO: Move MakeToken to TestUtil
	"time"
)

func toProtoByte32List(list *[][32]byte) *[]proto.Byte32 {
	newList := make([]proto.Byte32, 0)
	for _, element := range *list {
		newList = append(newList, (proto.Byte32)(element))
	}
	return &newList
}

func to32ByteList(list *[]proto.Byte32) *[][32]byte {
	newList := make([][32]byte, 0, 0)
	for _, element := range *list {
		newList = append(newList, ([32]byte)(element))
	}
	return &newList
}

func createAccount(conn *transport.Conn, inBuf []byte, outBuf []byte) error {
	command := &proto.ClientToServer{
		CreateAccount: protobuf.Bool(true),
	}
	if err := writeProtobuf(conn, outBuf, command); err != nil {
		return err
	}

	_, err := receiveProtobuf(conn, inBuf)
	if err != nil {
		return err
	}
	return nil
}

func listUserMessages(conn *transport.Conn, inBuf []byte, outBuf []byte) (*[][32]byte, error) {
	listMessages := &proto.ClientToServer{
		ListMessages: protobuf.Bool(true),
	}
	if err := writeProtobuf(conn, outBuf, listMessages); err != nil {
		return nil, err
	}

	response, err := receiveProtobuf(conn, inBuf)
	if err != nil {
		return nil, err
	}

	return to32ByteList(&response.MessageList), nil
}

func downloadEnvelope(conn *transport.Conn, inBuf []byte, outBuf []byte, messageHash *[32]byte) ([]byte, error) {
	getEnvelope := &proto.ClientToServer{
		DownloadEnvelope: (*proto.Byte32)(messageHash),
	}
	if err := writeProtobuf(conn, outBuf, getEnvelope); err != nil {
		return nil, err
	}

	response, err := receiveProtobuf(conn, inBuf)
	if err != nil {
		return nil, err
	}
	return response.Envelope, nil
}

func deleteMessages(conn *transport.Conn, inBuf []byte, outBuf []byte, messageList *[][32]byte) error {
	deleteMessages := &proto.ClientToServer{
		DeleteMessages: *toProtoByte32List(messageList),
	}
	if err := writeProtobuf(conn, outBuf, deleteMessages); err != nil {
		return err
	}

	_, err := receiveProtobuf(conn, inBuf)
	if err != nil {
		return err
	}
	return nil
}

func uploadKeys(conn *transport.Conn, inBuf []byte, outBuf []byte, keyList *[][]byte) error {
	uploadKeys := &proto.ClientToServer{
		UploadSignedKeys: *keyList,
	}
	if err := writeProtobuf(conn, outBuf, uploadKeys); err != nil {
		return nil
	}

	_, err := receiveProtobuf(conn, inBuf)
	if err != nil {
		return err
	}
	return nil
}

func getKey(conn *transport.Conn, inBuf []byte, outBuf []byte, pk *[32]byte) ([]byte, error) {
	getKey := &proto.ClientToServer{
		GetSignedKey: (*proto.Byte32)(pk),
	}
	if err := writeProtobuf(conn, outBuf, getKey); err != nil {
		return nil, err
	}

	response, err := receiveProtobuf(conn, inBuf)
	if err != nil {
		return nil, err
	}
	return response.SignedKey, nil
}

func getNumKeys(conn *transport.Conn, inBuf []byte, outBuf []byte, pk *[32]byte) (int64, error) {
	getNumKeys := &proto.ClientToServer{
		GetNumKeys: (*proto.Byte32)(pk),
	}
	if err := writeProtobuf(conn, outBuf, getNumKeys); err != nil {
		return 0, err
	}

	response, err := receiveProtobuf(conn, inBuf)
	if err != nil {
		return 0, err
	}
	return *response.NumKeys, nil
}

func enablePush(conn *transport.Conn, inBuf []byte, outBuf []byte) error {
	true_ := true
	command := &proto.ClientToServer{
		ReceiveEnvelopes: &true_,
	}
	if err := writeProtobuf(conn, outBuf, command); err != nil {
		return err
	}
	_, err := receiveProtobuf(conn, inBuf)
	if err != nil {
		return err
	}
	return nil
}

func uploadMessageToUser(conn *transport.Conn, inBuf []byte, outBuf []byte, pk *[32]byte, envelope []byte) error {
	message := &proto.ClientToServer_DeliverEnvelope{
		User:     (*proto.Byte32)(pk),
		Envelope: envelope,
	}
	deliverCommand := &proto.ClientToServer{
		DeliverEnvelope: message,
	}
	if err := writeProtobuf(conn, outBuf, deliverCommand); err != nil {
		return err
	}

	_, err := receiveProtobuf(conn, inBuf)
	if err != nil {
		return err
	}
	return nil
}

func writeProtobuf(conn *transport.Conn, outBuf []byte, message *proto.ClientToServer) error {
	size, err := message.MarshalTo(outBuf)
	if err != nil {
		return err
	}
	conn.WriteFrame(outBuf[:size])
	return nil
}

func receiveProtobuf(conn *transport.Conn, inBuf []byte) (*proto.ServerToClient, error) {
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

func denameCreateAccount(name []byte, config *client.Config) (*[32]byte, *client.Client, error) {
	newClient, err := client.NewClient(config, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	//TODO: All these names are horrible, please change them
	pkAuth, skAuth, err := box.GenerateKey(rand.Reader)

	chatProfile := &proto.Profile{
		ServerAddressTCP:  "",
		ServerTransportPK: (proto.Byte32)([32]byte{}),
		UserIDAtServer:    (proto.Byte32)([32]byte{}),
		KeySigningKey:     (proto.Byte32)([32]byte{}),
		MessageAuthKey:    (proto.Byte32)(*pkAuth),
	}

	chatProfileBytes, err := protobuf.Marshal(chatProfile)
	if err != nil {
		return nil, nil, err
	}

	profile, sk, err := client.NewProfile(nil, nil)
	if err != nil {
		return nil, nil, err
	}

	client.SetProfileField(profile, PROFILE_FIELD_ID, chatProfileBytes)

	err = newClient.Register(sk, name, profile, testutil2.MakeToken())
	if err != nil {
		return nil, nil, err
	}

	return skAuth, newClient, nil
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
