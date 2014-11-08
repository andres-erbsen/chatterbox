package main

//accept connection
//new thread!
//authenticate client
//get new cool connection
//make a new user with that connection
import (
	"bytes"
	"code.google.com/p/gogoprotobuf/io"
	"crypto/sha256"
	"fmt"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"net"
	"sync"
)

type Uid [32]byte
type Envelope []byte

type Server struct {
	database *leveldb.DB
	shutdown chan struct{}
	listener net.Listener
	wg       sync.WaitGroup
}

func StartServer(db *leveldb.DB, shutdown chan struct{}) (*Server, error) {
	//TODO: It's possible we want to call defer db.Close() here and not in the calling method
	listener, err := net.Listen("tcp", ":8888")
	if err != nil {
		return nil, err
	}
	server := &Server{
		database: db,
		shutdown: shutdown,
		listener: listener,
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
	fmt.Printf("wg %v\n", server.wg)
	return nil
}

func (server *Server) handleClientShutdown(connection net.Conn) {
	defer server.wg.Done()
clientShutdown:
	for {
		select {
		case <-server.shutdown:
			connection.Close()
			break clientShutdown
		default:
		}
	}
	return
}

//for each client, listen for commands
func (server *Server) handleClient(connection net.Conn) error {
	//return nil
	//err := server.database.Put([]byte("yolo"), []byte(""), nil)
	//return err
	defer server.wg.Done()
	uid, newConnection, err := authenticateClient(connection)
	if err != nil {
		return err
	}
	server.wg.Add(1)
	go server.handleClientShutdown(newConnection)

	reader := io.NewDelimitedReader(newConnection, 16*1024)
	writer := io.NewDelimitedWriter(newConnection)
	command := new(proto.ClientToServer)
clientCommands:
	for {
		if err := reader.ReadMsg(command); err != nil {
			select {
			case <-server.shutdown:
				break clientCommands
			default: //
			}
			return err
		}
		if command.CreateAccount != nil {
			if err := server.createAccount(newConnection, uid, writer); err != nil {
				return err
			}
		}
		if command.DeliverEnvelope != nil {
			user := *(*Uid)(command.DeliverEnvelope.User)
			envelope := command.DeliverEnvelope.Envelope
			if err := server.deliverEnvelope(newConnection, user, envelope, writer); err != nil {
				return err
			}
		}
		if command.ListMessages != nil {
			if err := server.listMessages(newConnection, uid, writer); err != nil {
				return err
			}
		}
	}
	return nil
}

func (server *Server) listMessages(connection net.Conn, user Uid, writer io.Writer) error {
	messages, err := server.getMessageList(user)
	if err != nil {
		messageList := &proto.ServerToClient{
			Status: proto.ServerToClient_PARSE_ERROR.Enum(),
		}
		if err := writer.WriteMsg(messageList); err != nil {
			return err
		}
		return err
	}
	messageList := &proto.ServerToClient{
		Status:      proto.ServerToClient_OK.Enum(),
		MessageList: messages,
	}
	return writer.WriteMsg(messageList)
}

func (server *Server) getMessageList(user Uid) ([][]byte, error) {
	messages := make([][]byte, 0, 64) //TODO: Reasonable starting cap for this buffer
	prefix := append([]byte{'m'}, user[:]...)
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
func (server *Server) deliverEnvelope(connection net.Conn, user Uid, envelope Envelope, writer io.Writer) error {
	if err := server.newMessage(user, envelope); err != nil {
		if err := server.writeResponse(writer, proto.ServerToClient_PARSE_ERROR.Enum()); err != nil {
			return err
		}
		return err
	}
	return server.writeResponse(writer, proto.ServerToClient_OK.Enum())
}

func (server *Server) createAccount(connection net.Conn, uid Uid, writer io.Writer) error {
	if err := server.newUser(uid); err != nil {
		fmt.Printf("Error")
		fmt.Printf("%v\n", err)
		if err := server.writeResponse(writer, proto.ServerToClient_PARSE_ERROR.Enum()); err != nil {
			return err
		}
		return err
	}
	return server.writeResponse(writer, proto.ServerToClient_OK.Enum())
}

func (server *Server) writeResponse(writer io.Writer, status *proto.ServerToClient_StatusCode) error {
	response := &proto.ServerToClient{
		Status: status,
	}
	return writer.WriteMsg(response)
}

func (server *Server) newMessage(uid Uid, envelope Envelope) error {
	// add message to the database
	messageHash := sha256.Sum256(envelope)
	key := append(append([]byte{'m'}, uid[:]...), messageHash[:]...)
	return server.database.Put(key, envelope[:], nil)
}

func (server *Server) newUser(uid Uid) error {
	// add user to the database
	err := server.database.Put(append([]byte{'u'}, uid[:]...), []byte(""), nil)
	return err
}

func authenticateClient(connection net.Conn) (Uid, net.Conn, error) {
	return [32]byte{}, connection, nil
}
