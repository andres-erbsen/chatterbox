// client daemon
//   watches the file system for new messages --> sends them
//   communicates with the server --> receive new messages
package daemon

import (
	"code.google.com/p/go.exp/fsnotify"
	"errors"
	util "github.com/andres-erbsen/chatterbox/client"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/dename/client"
	"log"
	"os"
	"time"
)

const MAX_MESSAGE_SIZE = 16 * 1024

func Start(rootDir string) error {
	conf := LoadConfig(&Config{
		RootDir:    rootDir,
		Now:        time.Now,
		TempPrefix: "daemon",
	})

	if err := InitFs(conf); err != nil {
		return err
	}

	inBuf := make([]byte, MAX_MESSAGE_SIZE)
	outBuf := make([]byte, MAX_MESSAGE_SIZE)

	ourDename := conf.Dename

	denameClient, err := client.NewClient(nil, nil, nil)
	if err != nil {
		return err
	}
	conf.denameClient = denameClient
	conf.ourDename = ourDename
	conf.inBuf = inBuf
	conf.outBuf = outBuf

	return nil
}

func Run(conf *Config, shutdown <-chan struct{}) error {
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

	ourConn, err := util.CreateServerConn(conf.ourDename, conf.denameClient)
	if err != nil {
		return err
	}

	notifies := make(chan []byte)
	replies := make(chan *proto.ServerToClient)

	connToServer := &util.ConnectionToServer{
		Buf:          conf.inBuf,
		Conn:         ourConn,
		ReadReply:    replies,
		ReadEnvelope: notifies,
	}

	go connToServer.ReceiveMessages()

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

						ourSkAuth := (*[32]byte)(&conf.MessageAuthSecretKey)
						var theirPk *[32]byte          //TODO: Load from file
						var encMsg, theirDename []byte //TODO: Load from file

						theirInBuf := make([]byte, MAX_MESSAGE_SIZE)

						theirConn, err := util.CreateServerConn(theirDename, conf.denameClient)
						if err != nil {
							return err
						}
						theirKey, err := util.GetKey(theirConn, theirInBuf, conf.outBuf, theirPk, theirDename, conf.denameClient)
						if err != nil {
							return err
						}
						encMsg, ratch, err := util.EncryptAuthFirst(theirDename, msg, ourSkAuth, theirKey, conf.denameClient)
						StoreRatchet(conf, string(theirDename), ratch)
						if err != nil {
							return err
						}
						err = util.UploadMessageToUser(theirConn, theirInBuf, conf.outBuf, theirPk, encMsg)
						if err != nil {
							return err
						}
					} else { //TODO: Not-first message in this conversation
						msg := []byte("Message") //TODO: msg is metadata + conversation

						var theirPk *[32]byte
						var encMsg, theirDename []byte

						msgRatch, err := LoadRatchet(conf, string(theirDename))
						if err != nil {
							return err
						}

						theirInBuf := make([]byte, MAX_MESSAGE_SIZE)

						theirConn, err := util.CreateServerConn(theirDename, conf.denameClient)
						if err != nil {
							return err
						}

						encMsg, ratch, err := util.EncryptAuth(theirDename, msg, msgRatch)
						StoreRatchet(conf, string(theirDename), ratch)

						if err != nil {
							return err
						}
						err = util.UploadMessageToUser(theirConn, theirInBuf, conf.outBuf, theirPk, encMsg)
						if err != nil {
							return err
						}
					}
				}
			}
		case envelope := <-connToServer.ReadEnvelope:
			if true { //TODO: is the first message we're receiving from the person
				var skList [][32]byte
				var skAuth *[32]byte
				ratch, msg, index, err := util.DecryptAuthFirst(envelope, skList, skAuth, conf.denameClient)
				message := new(proto.Message)
				if err := message.Unmarshal(msg); err != nil {
					return err
				}
				StoreRatchet(conf, string(message.Dename), ratch)
				//TODO: Delete skList form list
			} else {
				ratchets, err := AllRatchets(conf)
				if err != nil {
					return err
				}
				var ratch *ratchet.Ratchet
				var msg []byte
				for _, msgRatch := range ratchets {
					ratch, msg, err = util.DecryptAuth(envelope, msgRatch)
					if err == nil {
						break
					}
				}
				if msg == nil {
					return errors.New("Invalid message received.")
				}
				message := new(proto.Message)
				if err := message.Unmarshal(msg); err != nil {
					return err
				}
				StoreRatchet(conf, string(message.Dename), ratch)
				//TODO: Take out metadata + conversation from msg, then store

			}
		case err := <-watcher.Error:
			if err != nil {
				return err
			}
		}
	}

}
