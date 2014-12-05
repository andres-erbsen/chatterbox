package main

/*

import (
	"code.google.com/p/go.crypto/nacl/box"
	protobuf "code.google.com/p/gogoprotobuf/proto"
	"crypto/rand"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/transport"
	"net"
	"time"
)

func createAccount(conn *transport.Conn, inBuf []byte, outBuf []byte) error {
	command := &proto.ClientToServer{
		CreateAccount: protobuf.Bool(true),
	}
	if err := writeProtobuf(conn, outBuf, command, t); err != nil {
		return err
	}

	_, err := receiveProtobuf(conn, inBuf, t)
	if err != nil {
		return err
	}
}

func uploadMessageToUser(conn *transport.Conn, inBuf []byte, outBuf []byte, pk *[32]byte, envelope []byte) error {
	message := &proto.ClientToServer_DeliverEnvelope{
		User:     (*proto.Byte32)(pk),
		Envelope: envelope,
	}
	deliverCommand := &proto.ClientToServer{
		DeliverEnvelope: message,
	}

	if err := writeProtobuf(conn, outBuf, deliverCommand, t); err != nil {
		return err
	}

	_, err := receiveProtobuf(conn, inBuf, t)
	if err != nil {
		return err
	}
	return nil
}

func listUserMessages(conn *transport.Conn, inBuf []byte, outBuf []byte) (*[][32]byte, error) {
	listMessages := &proto.ClientToServer{
		ListMessages: protobuf.Bool(true),
	}
	if err := writeProtobuf(conn, outBuf, listMessages, t); err != nil {
		return nil, err
	}

	response, err := receiveProtobuf(conn, inBuf, t)
	if err != nil {
		return nil, err
	}

	return to32ByteList(&response.MessageList), nil
}

func downloadEnvelope(conn *transport.Conn, inBuf []byte, outBuf []byte, messageHash *[32]byte) ([]byte, error) {
	getEnvelope := &proto.ClientToServer{
		DownloadEnvelope: (*proto.Byte32)(messageHash),
	}
	if err := writeProtobuf(conn, outBuf, getEnvelope, t); err != nil {
		return nil, err
	}

	response, err := receiveProtobuf(conn, inBuf, t)
	if err != nil {
		return nil, err
	}
	return response.Envelope, nil
}

func deleteMessages(conn *transport.Conn, inBuf []byte, outBuf []byte, messageList *[][32]byte) error {
	deleteMessages := &proto.ClientToServer{
		DeleteMessages: *toProtoByte32List(messageList),
	}
	if err := writeProtobuf(conn, outBuf, deleteMessages, t); err != nil {
		return nil, err
	}

	_, err := receiveProtobuf(conn, inBuf, t)
	if err != nil {
		return err
	}
	return nil
}

func uploadKeys(conn *transport.Conn, inBuf []byte, outBuf []byte, keyList *[][32]byte) error {
	uploadKeys := &proto.ClientToServer{
		UploadKeys: *toProtoByte32List(keyList),
	}
	if err := writeProtobuf(conn, outBuf, uploadKeys, t); err != nil {
		return nil, err
	}

	_, err := receiveProtobuf(conn, inBuf, t)
	if err != nil {
		return err
	}
	return nil
}

func getKey(conn *transport.Conn, inBuf []byte, outBuf []byte, pk *[32]byte) (*[32]byte, error) {
	getKey := &proto.ClientToServer{
		GetKey: (*proto.Byte32)(pk),
	}
	if err := writeProtobuf(conn, outBuf, getKey, t); err != nil {
		return nil, err
	}

	response, err := receiveProtobuf(conn, inBuf, t)
	if err != nil {
		return nil, err
	}
	return (*[32]byte)(response.Key), nil
}

func getNumKeys(conn *transport.Conn, inBuf []byte, outBuf []byte, pk *[32]byte) (int64, error) {
	getNumKeys := &proto.ClientToServer{
		GetNumKeys: (*proto.Byte32)(pk),
	}
	if err := writeProtobuf(conn, outBuf, getNumKeys, t); err != nil {
		return nil, err
	}

	response, err := receiveProtobuf(conn, inBuf, t)
	if err != nil {
		return nil, err
	}
	return *response.NumKeys, nil
}

func enablePush(conn *transport.Conn, inBuf []byte, outBuf []byte) error {
	true_ := true
	command := &proto.ClientToServer{
		ReceiveEnvelopes: &true_,
	}
	if err := writeProtobuf(conn, outBuf, command, t); err != nil {
		return err
	}
	_, err := receiveProtobuf(conn, inBuf, t)
	if err != nil {
		return err
	}
	return nil
}

func dropMessage(server *Server, uid *[32]byte, message []byte) error {
	oldConn, err := net.Dial("tcp", server.listener.Addr().String())
	if err != nil {
		return err
	}

	pkp, skp, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	conn, _, err := transport.Handshake(oldConn, pkp, skp, nil, MAX_MESSAGE_SIZE)
	if err != nil {
		return err
	}

	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	if err := uploadMessageToUser(conn, inBuf, outBuf, t, uid, message); err != nil {
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
	if err := writeProtobuf(conn, outBuf, deliverCommand, t); err != nil {
		return err
	}

	_, err := receiveProtobuf(conn, inBuf, t)
	if err != nil {
		return err
	}
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
*/
