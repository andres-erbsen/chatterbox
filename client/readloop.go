package client

import (
	"sync"

	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/transport"
)

type EnvelopeWithId struct {
	Envelope []byte
	Id       *[32]byte
}

type ConnectionToServer struct {
	InBuf        []byte
	Conn         *transport.Conn
	ReadReply    chan *proto.ServerToClient // TODO: do we want to return an error?
	ReadEnvelope chan *EnvelopeWithId

	Shutdown     chan struct{}
	WaitShutdown sync.WaitGroup
}

func (c *ConnectionToServer) ReceiveMessages() error {
	c.WaitShutdown.Add(1)
	go func() {
		defer c.WaitShutdown.Done()
		<-c.Shutdown
		c.Conn.Close()
	}()
	for {
		msg, err := ReceiveProtobuf(c.Conn, c.InBuf)
		select {
		case <-c.Shutdown:
			return nil
		default:
			if err != nil {
				return err
			}
		}
		if msg.Envelope != nil {
			envwithid := &EnvelopeWithId{
				Envelope: msg.Envelope,
				Id:       (*[32]byte)(msg.MessageId),
			}
			go func() { c.ReadEnvelope <- envwithid }() // TODO: bounded buffer?
		} else {
			c.ReadReply <- msg
		}
	}
}

func (conn *ConnectionToServer) WriteProtobuf(msg *proto.ClientToServer) error {
	return WriteProtobuf(conn.Conn, msg)
}
