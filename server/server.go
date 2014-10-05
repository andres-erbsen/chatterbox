package main

//accept connection
//new thread!
//authenticate client
//get new cool connection
//make a new user with that connection
import (
	"code.google.com/p/gogoprotobuf/io"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"net"
)

type Uid [32]byte
type Envelope struct{}

type Server struct {
	database *leveldb.DB
	shutdown chan struct{}
}

func RunServer(db *leveldb.DB, shutdown chan struct{}) error {
	//TODO: It's possible we want to call defer db.Close() here and not in the calling method
	server := &Server{
		database: db,
		shutdown: shutdown,
	}
	listener, err := net.Listen("tcp", ":8888")
	if err != nil {
		return err
	}
	defer listener.Close()
	go server.listenForServerShutdown(listener)
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go server.handleClient(conn)

	}
}

func (server *Server) listenForServerShutdown(listener net.Listener) {
	for _ = range server.shutdown {
		listener.Close()
	}
}

func (server *Server) listenForClientShutdown(connection net.Conn) {
	for _ = range server.shutdown {
		connection.Close()
	}
}

//for each client, listen for commands
func (server *Server) handleClient(connection net.Conn) error {
	uid, newConnection, err := authenticateClient(connection)
	if err != nil {
		return err
	}
	defer newConnection.Close()
	go server.listenForClientShutdown(newConnection)
	reader := io.NewDelimitedReader(newConnection, 16*1024)
	writer := io.NewDelimitedWriter(newConnection)
	command := new(Messages.ClientToServer)
	for {
		if err := reader.ReadMsg(command); err != nil {
			return err
		}
		if command.CreateAccount != nil {
			if err := server.createAccount(newConnection, uid, writer); err != nil {
				return err
			}
		}
	}
}

func (server *Server) createAccount(connection net.Conn, uid Uid, writer io.Writer) error {
	if err := server.newUser(connection, uid); err != nil {
		if err := server.writeResponse(writer, Messages.ServerToClient_PARSE_ERROR.Enum()); err != nil {
			return err
		}
		return err
	}
	return server.writeResponse(writer, Messages.ServerToClient_OK.Enum())
}

func (Server *Server) writeResponse(writer io.Writer, status *Messages.ServerToClient_StatusCode) error {
	response := &Messages.ServerToClient{
		Status: status,
	}
	return writer.WriteMsg(response)
}

func (server *Server) newUser(connection net.Conn, uid Uid) error {
	// add user to the database
	return server.database.Put(append([]byte{'u'}, uid[:]...), []byte(""), nil)
}

func authenticateClient(connection net.Conn) (Uid, net.Conn, error) {
	return [32]byte{}, connection, nil
}
