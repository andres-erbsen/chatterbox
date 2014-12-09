package client

/*
import (
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/transport"
	"sync"
)

type connectionToServer struct {
	buf          []byte
	conn         *transport.Conn
	ReadReply    chan *proto.ServerToClient // TODO: do we want to return an error?
	ReadEnvelope chan []byte

	suhtdown     chan struct{}
	waitShutdown sync.WaitGroup
}

func (c *connectionToServer) receiveMessages() error {
	c.waitShutdown.Add(1)
	go func() {
		defer c.waitShutdown.Done()
		<-c.shutdown
		c.conn.Close()
	}()
	for {
		msg, err := receiveProtobuf(c.conn, c.buf)
		select {
		case <-c.shutdown:
			return nil
		default:
			if err != nil {
				return err
			}
		}
		if msg.Envelope != nil {
			c.ReadEnvelope <- msg.Evelope
		} else {
			c.ReadReply <- msg
		}
	}
}
*/
