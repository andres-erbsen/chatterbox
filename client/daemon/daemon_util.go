package main

import (
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"errors"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/transport"
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

func uploadKeys(conn *transport.Conn, inBuf []byte, outBuf []byte, keyList *[][32]byte) error {
	uploadKeys := &proto.ClientToServer{
		UploadKeys: *toProtoByte32List(keyList),
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

func getKey(conn *transport.Conn, inBuf []byte, outBuf []byte, pk *[32]byte) (*[32]byte, error) {
	getKey := &proto.ClientToServer{
		GetKey: (*proto.Byte32)(pk),
	}
	if err := writeProtobuf(conn, outBuf, getKey); err != nil {
		return nil, err
	}

	response, err := receiveProtobuf(conn, inBuf)
	if err != nil {
		return nil, err
	}
	return (*[32]byte)(response.Key), nil
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
