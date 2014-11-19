package main

//TODO: Ask Andres about key acceptance protocol
//accept connection
//new thread!
//authenticate client
//get new cool connection
//make a new user with that connection
import (
	//"code.google.com/p/gogoprotobuf/io"
	"crypto/sha256"
	"errors"
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

type Server struct {
	database *leveldb.DB
	shutdown chan struct{}
	listener net.Listener
	notifier Notifier
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
		go server.handleClient(&conn)
	}
	return nil
}

func (server *Server) handleClientShutdown(connection *transport.Conn) {
	defer server.wg.Done()
	<-server.shutdown
	(*connection).Close()
	return
}

// readClientCommands reads client commands from a connnection and sends them
// to channel commands. On error, the error is sent to channel disconnect and
// both channels (but not the connection are closed).
// commands is a TWO-WAY channel! the reader must reach return each cmd after
// interpreting it, readClientCommands will call cmd.Reset() and reuse it.
func (server *Server) readClientCommands(conn *transport.Conn,
	commands chan *proto.ClientToServer, disconnected chan error) {
	defer server.wg.Done()
	defer close(commands)
	defer close(disconnected)
	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	cmd := new(proto.ClientToServer)
	for {
		num, err := conn.ReadFrame(inBuf)
		if err != nil {
			disconnected <- err
			return
		}
		if err := cmd.Unmarshal(inBuf[:num]); err != nil {
			disconnected <- err
			return
		}
		commands <- cmd
		cmd = <-commands
		cmd.Reset()
	}
}

// readClientNotifications is a for loop of blocking reads on notificationsIn
// and non-blocking sends on notificationsOut. If the input channel is closed
// or a send would block, the output channel is closed.
func (server *Server) readClientNotifications(notificationsIn chan []byte, notificationsOut chan []byte) {
	var hasOverflowed bool
	defer server.wg.Done()
	for n := range notificationsIn {
		if !hasOverflowed {
			select {
			case notificationsOut <- n:
			default:
				hasOverflowed = true
				close(notificationsOut)
			}
		}
	}
}

//for each client, listen for commands
func (server *Server) handleClient(connection *net.Conn) error {
	defer server.wg.Done()
	newConnection, uid, err := transport.Handshake(*connection, server.pk, server.sk, nil, MAX_MESSAGE_SIZE) //TODO: Decide on this bound
	if err != nil {
		return err
	}

	commands := make(chan *proto.ClientToServer)
	disconnected := make(chan error)
	server.wg.Add(2)
	go server.readClientCommands(newConnection, commands, disconnected)
	go server.handleClientShutdown(newConnection)

	var notificationsUnbuffered, notifications chan []byte
	var notifyEnabled bool

	outBuf := make([]byte, MAX_MESSAGE_SIZE)
	response := new(proto.ServerToClient)
	for {
		select {
		case err := <-disconnected:
			return err
		case cmd := <-commands:
			if cmd.CreateAccount != nil && *cmd.CreateAccount != false {
				err = server.newUser(uid)
			} else if cmd.DeliverEnvelope != nil {
				err = server.newMessage((*[32]byte)(cmd.DeliverEnvelope.User),
					cmd.DeliverEnvelope.Envelope)
			} else if cmd.ListMessages != nil && *cmd.ListMessages != false {
				response.MessageList, err = server.getMessageList(uid)
			} else if cmd.DownloadEnvelope != nil {
				response.Envelope, err = server.getEnvelope(uid, cmd.DownloadEnvelope)
			} else if cmd.DeleteMessages != nil {
				messageList := cmd.DeleteMessages
				err = server.deleteMessages(uid, &messageList)
			} else if cmd.UploadKeys != nil {
				err = server.newKeys(uid, cmd.UploadKeys)
			} else if cmd.GetKey != nil {
				response.Key, err = server.getKey((*[32]byte)(cmd.GetKey))
			} else if cmd.ReceiveEnvelopes != nil {
				if *cmd.ReceiveEnvelopes && !notifyEnabled {
					notifyEnabled = true
					notificationsUnbuffered = server.notifier.StartWaiting(uid)
					notifications = make(chan []byte)
					server.wg.Add(1)
					go server.readClientNotifications(notificationsUnbuffered, notifications)
				} else if !*cmd.ReceiveEnvelopes && notifyEnabled {
					server.notifier.StopWaitingSync(uid, notificationsUnbuffered)
					notifyEnabled = false
				}
			}
			if err != nil {
				response.Status = proto.ServerToClient_PARSE_ERROR.Enum()
			} else {
				response.Status = proto.ServerToClient_OK.Enum()
			}
			if err = server.writeProtobuf(newConnection, outBuf, response); err != nil {
				return err
			}
			commands <- cmd
		case notification, ok := <-notifications:
			if !notifyEnabled {
				continue
			}
			if !ok {
				notifyEnabled = false
				go server.notifier.StopWaitingSync(uid, notificationsUnbuffered)
				continue
			}
			response.Envelope = notification
			response.Status = proto.ServerToClient_OK.Enum()
			if err = server.writeProtobuf(newConnection, outBuf, response); err != nil {
				return err
			}
		}
		response.Reset()
	}
	return nil
}

func (server *Server) deleteKey(uid *[32]byte, key []byte) error {
	keyHash := sha256.Sum256(key)
	dbKey := append(append([]byte{'k'}, uid[:]...), keyHash[:]...)
	return server.database.Delete(dbKey, nil)
}

func (server *Server) getKey(user *[32]byte) ([]byte, error) {
	// TODO: synchronization. Two concurrent gets MUST get different keys
	// unless it is the last one.
	prefix := append([]byte{'k'}, (*user)[:]...)
	keyRange := util.BytesPrefix(prefix)
	iter := server.database.NewIterator(keyRange, nil)
	defer iter.Release()
	if iter.First() == false {
		return nil, errors.New("No keys left in database")
	}
	err := iter.Error()
	key := iter.Value()
	server.deleteKey(user, key)
	return key, err
}

func (server *Server) newKeys(uid *[32]byte, keyList [][]byte) error {
	for _, key := range keyList {
		keyHash := sha256.Sum256(key)
		dbKey := append(append([]byte{'k'}, uid[:]...), keyHash[:]...)
		err := server.database.Put(dbKey, key, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
func (server *Server) deleteMessages(uid *[32]byte, messageList *[][]byte) error {
	for _, messageHash := range *messageList {
		key := append(append([]byte{'m'}, uid[:]...), messageHash[:]...)
		err := server.database.Delete(key, nil)
		if err != nil { //TODO: Ask Andres if there was a better way to do this
			return err
		}
	}
	return nil
}

func (server *Server) getEnvelope(uid *[32]byte, messageHash []byte) ([]byte, error) {
	key := append(append([]byte{'m'}, uid[:]...), (messageHash)[:]...)
	envelope, err := server.database.Get(key, nil)
	return envelope, err
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
	messages := make([][]byte, 0)
	prefix := append([]byte{'m'}, (*user)[:]...)
	messageRange := util.BytesPrefix(prefix)
	iter := server.database.NewIterator(messageRange, nil)
	for iter.Next() {
		message := append([]byte{}, iter.Key()[len(prefix):]...)
		messages = append(messages, message)
	}
	err := iter.Error()
	iter.Release()
	return messages, err
}

func (server *Server) newMessage(uid *[32]byte, envelope []byte) error {
	// add message to the database
	messageHash := sha256.Sum256(envelope)
	key := append(append([]byte{'m'}, uid[:]...), messageHash[:]...)
	err := server.database.Put(key, (envelope)[:], nil)
	if err != nil {
		return err
	}
	server.notifier.Notify(uid, envelope)
	return nil
}

func (server *Server) newUser(uid *[32]byte) error {
	return server.database.Put(append([]byte{'u'}, uid[:]...), []byte(""), nil)
}
