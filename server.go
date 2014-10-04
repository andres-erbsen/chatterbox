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
}

func RunServer() error {
	db, err := leveldb.OpenFile("github.com/andres-erbsen/chatterbox/database", nil)
	if err != nil {
		return err
	}
	defer db.Close()
	server := &Server{
		database: db,
	}
	listener, err := net.Listen("tcp", ":8888")
	if err != nil {
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go server.handleClient(conn)
	}
}

//for each client, listen for commands
func (server *Server) handleClient(connection net.Conn) error {
	uid, newConnection, err := server.authenticateClient(connection)
	if err != nil {
		return err
	}
	defer newConnection.Close()
	reader := io.NewDelimitedReader(newConnection, 16*1024)
	command := new(Messages.ClientToServer)
	for {
		if err := reader.ReadMsg(command); err != nil {
			return err
		}
		if command.CreateAccount != nil {
			server.newUser(newConnection, uid)
		}
	}
}

func (server *Server) newUser(connection net.Conn, uid Uid) error {
	// add user to the database
	return server.database.Put(append([]byte{'u'}, uid[:]...), []byte(""), nil)
}

func (server *Server) authenticateClient(connection net.Conn) (Uid, net.Conn, error) {
	return [32]byte{}, connection, nil
}
