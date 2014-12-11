// client daemon
//   watches the file system for new messages --> sends them
//   communicates with the server --> receive new messages
package daemon

import (
	"code.google.com/p/go.exp/fsnotify"
	util "github.com/andres-erbsen/chatterbox/client"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	//"github.com/andres-erbsen/chatterbox/transport"
	"github.com/andres-erbsen/dename/client"
	"log"
	//"net"
	"os"
	"time"
)

const (
	MAX_MESSAGE_SIZE = 16 * 1024
	MAX_PREKEYS      = 100 //TODO make this configurable
	MIN_PREKEYS      = 50
)

func Run(rootDir string, shutdown <-chan struct{}) error {
	conf := LoadConfig(&Config{
		RootDir:    rootDir,
		Now:        time.Now,
		TempPrefix: "daemon",
	})

	// ensure that we have a correct directory structure
	// including a correctly-populated outbox
	if err := InitFs(conf); err != nil {
		return err
	}

	//// connect to the server
	//plainConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", *serverAddress, *serverPort))
	//conn, _, err := transport.Handshake(plainConn,
	//	(*[32]byte)(&publicProfile.UserIDAtServer),
	//	(*[32]byte)(&secretConfig.TransportSecretKeyForServer),
	//	&serverTransportPubkey, client.MAX_MESSAGE_SIZE)
	//if err != nil {
	//	return err
	//}
	//inBuf := make([]byte, client.MAX_MESSAGE_SIZE)
	//outBuf := make([]byte, client.MAX_MESSAGE_SIZE)

	//// load prekeys and ensure that we have enough of them
	//prekeys, err := LoadPrekeys(conf)
	//if err != nil {
	//	return err
	//}
	//numKeys, err := client.GetNumKeys(conn, inBuf, outBuf)
	//if err != nil {
	//	return err
	//}
	//if numKeys < MIN_KEYS {
	//	publicPrekeys, newSecretPrekeys, err := GeneratePrekeys(MAX_PREKEYS - numKeys)
	//	prekeys = Append(prekeys, newSecretPrekeys)
	//	err = client.UploadKeys(conn, inBuf, outBuf, publicPrekeys)
	//	if err != nil {
	//		return err // TODO handle this nicely
	//	}
	//}

	// set up a filesystem watcher on the outbox
	initFn := func(path string, f os.FileInfo, err error) error {
		log.Printf("init path: %s\n", path)
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = WatchDir(watcher, conf.OutboxDir(), initFn)
	if err != nil {
		return err
	}

	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	var config *client.Config //TODO: Load from file
	var ourDename []byte      //TODO: Load from file

	ourConn, err := util.CreateServerConn(ourDename, config)
	if err != nil {
		return err
	}

	notifies := make(chan []byte)
	replies := make(chan *proto.ServerToClient)

	connectionToServer := &util.ConnectionToServer{
		Buf:          inBuf,
		Conn:         ourConn,
		ReadReply:    replies,
		ReadEnvelope: notifies,
		Shutdown:     shutdown,
	}

	go connectionToServer.ReceiveMessages()

	for {
		select {
		case <-shutdown:
			return nil
		case ev := <-watcher.Event:
			// event in the directory structure; watch any new directories
			if !(ev.IsDelete() || ev.IsRename()) {
				err = WatchDir(watcher, ev.Name, initFn)
				if err != nil {
					return err
				}
				if true { //TODO: Fill in something's changed in the outbox
					if true { //TODO: First message in this conversation
						msg := []byte("Message") //TODO: msg is metadata + conversation

						var theirPk, theirSkAuth, theirPkSig *[32]byte //TODO: Load from file
						var encMsg, theirDename []byte                 //TODO: Load from file

						theirInBuf := make([]byte, MAX_MESSAGE_SIZE)

						theirConn, err := util.CreateServerConn(theirDename, config)
						if err != nil {
							return err
						}
						theirKey, err := util.GetKey(theirConn, theirInBuf, outBuf, theirPk, theirPkSig)
						if err != nil {
							return err
						}
						encMsg, ratch, err := util.EncryptAuthFirst(theirDename, msg, theirSkAuth, theirKey, config)
						ratch = ratch //TODO: Remove this line
						if err != nil {
							return err
						}
						err = util.UploadMessageToUser(theirConn, theirInBuf, outBuf, theirPk, encMsg)
						if err != nil {
							return err
						}
					} else { //TODO: Not-first message in this conversation
						var theirPk *[32]byte
						var config *client.Config
						var encMsg, theirDename []byte
						var msgRatch *ratchet.Ratchet

						theirInBuf := make([]byte, MAX_MESSAGE_SIZE)

						msg := []byte("Message") //TODO: msg is metadata + conversation
						theirConn, err := util.CreateServerConn(theirDename, config)
						if err != nil {
							return err
						}

						encMsg, ratch, err := util.EncryptAuth(theirDename, msg, msgRatch)
						ratch = ratch //TODO: Remove this line
						if err != nil {
							return err
						}
						err = util.UploadMessageToUser(theirConn, theirInBuf, outBuf, theirPk, encMsg)
						if err != nil {
							return err
						}
					}
				}
			}
		case err := <-watcher.Error:
			if err != nil {
				return err
			}
		}
	}

}
