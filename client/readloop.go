package client

import (
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/transport"
	"sync"
)

type ConnectionToServer struct {
	Buf          []byte
	Conn         *transport.Conn
	ReadReply    chan *proto.ServerToClient // TODO: do we want to return an error?
	ReadEnvelope chan []byte

	Shutdown     <-chan struct{}
	waitShutdown sync.WaitGroup
}

func (c *ConnectionToServer) ReceiveMessages() error {
	c.waitShutdown.Add(1)
	go func() {
		defer c.waitShutdown.Done()
		<-c.Shutdown
		c.Conn.Close()
	}()
	for {
		msg, err := ReceiveProtobuf(c.Conn, c.Buf)
		select {
		case <-c.Shutdown:
			return nil
		default:
			if err != nil {
				return err
			}
		}
		if msg.Envelope != nil {
			c.ReadEnvelope <- msg.Envelope
		} else {
			c.ReadReply <- msg
		}
	}
}
