package main

//accept connection
//new thread!
//authenticate client
//get new cool connection
//make a new user with that connection
import (
	"bytes"
	//"code.google.com/p/gogoprotobuf/io"
	"crypto/sha256"
	"fmt"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/transport"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"net"
	"sync"
)

//
//TODO: Check that sent messages go to user that exists
const MAX_MESSAGE_SIZE = 16 * 1024

type Envelope []byte

type Server struct {
	database *leveldb.DB
	shutdown chan struct{}
	listener net.Listener
	wg       sync.WaitGroup
	pk       *[32]byte
	sk       *[32]byte
}

var _ = fmt.Printf

func StartServer(db *leveldb.DB, shutdown chan struct{}, pk *[32]byte, sk *[32]byte) (*Server, error) {
	//TODO: It's possible we want to call defer db.Close() here and not in the calling method
	listener, err := net.Listen("tcp", ":8888")
	if err != nil {
		return nil, err
	}
	server := &Server{
		database: db,
		shutdown: shutdown,
		listener: listener,
		pk:       pk,
		sk:       sk,
	}
	server.wg.Add(1)
	go server.RunServer()
	//err = server.database.Put([]byte("yolo"), []byte(""), nil)
	return server, nil
}

func (server *Server) StopServer() {
	close(server.shutdown)
	server.listener.Close()
	server.wg.Wait()
}

func (server *Server) RunServer() error {
	//server.writeResponse(writer, proto.ServerToClient_OK.Enum())
	defer server.wg.Done()
serverLoop:
	for {
		select {
		case <-server.shutdown:
			break serverLoop
		default: //
		}
		conn, err := server.listener.Accept()
		if err != nil {
			return err
		}

		server.wg.Add(1)
		go server.handleClient(conn)
	}
	return nil
}

func (server *Server) handleClientShutdown(connection *transport.Conn) {
	defer server.wg.Done()
	<-server.shutdown
	connection.Close()
	return
}

//for each client, listen for commands
func (server *Server) handleClient(connection net.Conn) error {
	//return nil
	//err := server.database.Put([]byte("yolo"), []byte(""), nil)
	//return err
	defer server.wg.Done()
	newConnection, uid, err := transport.Handshake(connection, server.pk, server.sk, nil, MAX_MESSAGE_SIZE) //TODO: Decide on this bound
	if err != nil {
		return err
	}
	server.wg.Add(1)
	go server.handleClientShutdown(newConnection)

	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)
	//	reader := io.NewDelimitedReader(newConnection, 16*1024)
	//writer := io.NewDelimitedWriter(newConnection)
	command := new(proto.ClientToServer)
	response := new(proto.ServerToClient)
clientCommands:
	for {
		num, err := newConnection.ReadFrame(inBuf)
		//fmt.Printf("R %x\n", inBuf[:num])

		if err != nil {
			select {
			case <-server.shutdown:
				break clientCommands
			default: //
			}
			return err
		}
		if err := command.Unmarshal(inBuf[:num]); err != nil {
			return err
		}
		if command.CreateAccount != nil {
			err = server.newUser(uid)
		} else if command.DeliverEnvelope != nil {
			user := (*[32]byte)(command.DeliverEnvelope.User)
			envelope := command.DeliverEnvelope.Envelope
			err = server.newMessage(user, envelope)
		} else if command.ListMessages != nil {
			response.MessageList, err = server.getMessageList(uid)
		}

		if err != nil {
			response.Status = proto.ServerToClient_PARSE_ERROR.Enum()
		} else {
			response.Status = proto.ServerToClient_OK.Enum()
		}
		if err = server.writeProtobuf(newConnection, outBuf, response); err != nil {
			return err
		}
		command.Reset()
		response.Reset()
	}
	return nil
}

func (server *Server) writeProtobuf(conn *transport.Conn, outBuf []byte, message *proto.ServerToClient) error {
	size, err := message.MarshalTo(outBuf)
	if err != nil {
		return err
	}
	conn.WriteFrame(outBuf[:size])
	return nil
}

func (server *Server) getMessageList(user *[32]byte) ([][]byte, error) {
	messages := make([][]byte, 0, 64) //TODO: Reasonable starting cap for this buffer
	prefix := append([]byte{'m'}, (*user)[:]...)
	messageRange := util.BytesPrefix(prefix)
	iter := server.database.NewIterator(messageRange, nil)
	for iter.Next() {
		message := append([]byte{}, bytes.TrimPrefix(iter.Key(), prefix)...)
		messages = append(messages, message)
	}
	iter.Release()
	err := iter.Error()
	return messages, err
}

func (server *Server) newMessage(uid *[32]byte, envelope Envelope) error {
	// add message to the database
	messageHash := sha256.Sum256(envelope)
	key := append(append([]byte{'m'}, (*uid)[:]...), messageHash[:]...)
	return server.database.Put(key, envelope[:], nil)
}

func (server *Server) newUser(uid *[32]byte) error {
	// add user to the database
	err := server.database.Put(append([]byte{'u'}, (*uid)[:]...), []byte(""), nil)
	return err
}

//func authenticateClient(connection transport.Conn) ([32]byte, transport.Conn, error) {
//
//	return
//	[32]byte{}, connection, nil
//}
