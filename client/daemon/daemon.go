// client daemon
//   watches the file system for new messages --> sends them
//   communicates with the server --> receive new messages
package daemon

import (
	"code.google.com/p/go.exp/fsnotify"
	"errors"
	"fmt"
	util "github.com/andres-erbsen/chatterbox/client"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/dename/client"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

const (
	MAX_PREKEYS = 100 //TODO make this configurable
	MIN_PREKEYS = 50
)

func Start(rootDir string) (*Config, error) {
	conf := LoadConfig(&Config{
		RootDir:    rootDir,
		Now:        time.Now,
		TempPrefix: "daemon",
	})

	// ensure that we have a correct directory structure
	// including a correctly-populated outbox
	if err := InitFs(conf); err != nil {
		return nil, err
	}
	inBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)
	outBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)

	denameClient, err := client.NewClient(nil, nil, nil)
	if err != nil {
		return nil, err
	}
	conf.denameClient = denameClient
	conf.inBuf = inBuf
	conf.outBuf = outBuf

	return conf, nil
}

func (conf *Config) sendFirstMessage(msg []byte, theirDename []byte) (*ratchet.Ratchet, error) {
	//If using TOR, dename client is fresh TOR connection
	profile, err := conf.denameClient.Lookup(theirDename)
	if err != nil {
		return nil, err
	}

	chatProfileBytes, err := client.GetProfileField(profile, util.PROFILE_FIELD_ID)
	if err != nil {
		return nil, err
	}

	chatProfile := new(proto.Profile)
	if err := chatProfile.Unmarshal(chatProfileBytes); err != nil {
		return nil, err
	}

	addr := chatProfile.ServerAddressTCP
	pkSig := (*[32]byte)(&chatProfile.KeySigningKey)
	port := (int)(chatProfile.ServerPortTCP)
	pkTransport := (*[32]byte)(&chatProfile.ServerTransportPK)
	theirPk := (*[32]byte)(&chatProfile.UserIDAtServer)

	ourSkAuth := (*[32]byte)(&conf.MessageAuthSecretKey)

	theirInBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)

	theirConn, err := util.CreateForeignServerConn(theirDename, conf.denameClient, addr, port, pkTransport)
	if err != nil {
		return nil, err
	}
	theirKey, err := util.GetKey(theirConn, theirInBuf, conf.outBuf, theirPk, theirDename, pkSig)
	if err != nil {
		return nil, err
	}
	encMsg, ratch, err := util.EncryptAuthFirst(msg, ourSkAuth, theirKey, conf.denameClient)
	if err != nil {
		return nil, err
	}
	err = util.UploadMessageToUser(theirConn, theirInBuf, conf.outBuf, theirPk, encMsg)
	if err != nil {
		return nil, err
	}
	return ratch, nil
}

func (conf *Config) sendMessage(msg []byte, theirDename []byte, msgRatch *ratchet.Ratchet) (*ratchet.Ratchet, error) {
	//If using TOR, dename client is fresh TOR connection
	profile, err := conf.denameClient.Lookup(theirDename)
	if err != nil {
		return nil, err
	}

	chatProfileBytes, err := client.GetProfileField(profile, util.PROFILE_FIELD_ID)
	if err != nil {
		return nil, err
	}

	chatProfile := new(proto.Profile)
	if err := chatProfile.Unmarshal(chatProfileBytes); err != nil {
		return nil, err
	}

	addr := chatProfile.ServerAddressTCP
	port := (int)(chatProfile.ServerPortTCP)
	pkTransport := (*[32]byte)(&chatProfile.ServerTransportPK)
	theirPk := (*[32]byte)(&chatProfile.UserIDAtServer)

	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	theirInBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)

	theirConn, err := util.CreateForeignServerConn(theirDename, conf.denameClient, addr, port, pkTransport)

	if err != nil {
		return nil, err
	}

	encMsg, ratch, err := util.EncryptAuth(msg, msgRatch)
	if err != nil {
		return nil, err
	}
	err = util.UploadMessageToUser(theirConn, theirInBuf, conf.outBuf, theirPk, encMsg)
	if err != nil {
		return nil, err
	}
	return ratch, nil
}

func (conf *Config) decryptFirstMessage(envelope []byte, pkList []*[32]byte, skList []*[32]byte) (*proto.Message, *ratchet.Ratchet, int, error) {
	skAuth := (*[32]byte)(&conf.MessageAuthSecretKey)
	ratch, msg, index, err := util.DecryptAuthFirst(envelope, pkList, skList, skAuth, conf.denameClient)

	if err != nil {
		return nil, nil, -1, err
	}
	message := new(proto.Message)
	if err := message.Unmarshal(msg); err != nil {
		return nil, nil, -1, err
	}

	return message, ratch, index, nil
}

func (conf *Config) decryptMessage(envelope []byte, ratchets []*ratchet.Ratchet) (*proto.Message, *ratchet.Ratchet, error) {
	var ratch *ratchet.Ratchet
	var msg []byte
	for _, msgRatch := range ratchets {
		var err error
		ratch, msg, err = util.DecryptAuth(envelope, msgRatch)
		if err == nil {
			break // found the right ratchet
		}
	}
	if msg == nil {
		return nil, nil, errors.New("Invalid message received.")
	}
	message := new(proto.Message)
	if err := message.Unmarshal(msg); err != nil {
		return nil, nil, err
	}
	return message, ratch, nil
}

func processOutboxDir(conf *Config, dirname string) error {
	// parse metadata
	metadataFile := filepath.Join(dirname, MetadataFileName)
	if _, err := os.Stat(metadataFile); err != nil {
		return nil // no metadata --> not an outgoing message
	}

	metadata := proto.ConversationMetadata{}
	err := UnmarshalFromFile(metadataFile, &metadata)
	if err != nil {
		return err
	}

	// add ourselves to the participants list
	allParticipants := append(metadata.Participants, conf.Dename)

	// load messages
	potentialMessages, err := ioutil.ReadDir(dirname)
	if err != nil {
		return err
	}
	messages := make([][]byte, 0, len(potentialMessages))
	for _, finfo := range potentialMessages {
		if !finfo.IsDir() && finfo.Name() != MetadataFileName {
			msg, err := ioutil.ReadFile(filepath.Join(dirname, finfo.Name()))
			if err != nil {
				return err
			}

			// make protobuf for message; append it
			payload := proto.Message{
				Subject:      metadata.Subject,
				Participants: allParticipants,
				Dename:       conf.Dename,
				Contents:     msg,
			}
			payloadBytes, err := payload.Marshal()
			if err != nil {
				return err
			}
			messages = append(messages, payloadBytes)
		}
	}
	if len(messages) == 0 {
		return nil // no messages to send, just the metadata file
	}

	for _, recipient := range metadata.Participants {
		for _, msg := range messages {
			// replace msg with message+metadata (new protobuf?)
			fillAuth := util.FillAuthWith((*[32]byte)(&conf.MessageAuthSecretKey))
			checkAuth := util.CheckAuthWith(conf.denameClient)
			if err != nil {
				return err
			}
			if msgRatch, err := LoadRatchet(conf, recipient, fillAuth, checkAuth); err != nil { //First message in this conversation
				ratch, err := conf.sendFirstMessage(msg, recipient)
				if err != nil {
					return err
				}
				StoreRatchet(conf, recipient, ratch)
			} else { // Not-first message in this conversation
				ratch, err := conf.sendMessage(msg, recipient, msgRatch)
				if err != nil {
					return err
				}
				StoreRatchet(conf, recipient, ratch)
			}
		}
	}

	// copy the metadata file to the oconversation folder if it doesn't already exist
	convName, err := filepath.Rel(conf.OutboxDir(), dirname)
	if err != nil {
		return err
	}
	convPath := filepath.Join(conf.ConversationDir(), convName)
	if os.Mkdir(convPath, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	convMetadataFile := filepath.Join(convPath, MetadataFileName)
	_, err = os.Stat(convMetadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			if err = Copy(filepath.Join(dirname, MetadataFileName), convMetadataFile, 0600); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// move the sent messages to the conversation folder
	for _, finfo := range potentialMessages {
		if !finfo.IsDir() && finfo.Name() != MetadataFileName {
			if err = os.Rename(filepath.Join(dirname, finfo.Name()), filepath.Join(convPath, finfo.Name())); err != nil {
				// TODO handle os.IsExists(err)
				return err
			}
		}
	}

	return nil
}

func Run(conf *Config, shutdown <-chan struct{}) error {

	profile, err := LoadPublicProfile(conf)
	if err != nil {
		return err
	}

	ourConn, err := util.CreateHomeServerConn(
		conf.ServerAddressTCP, (*[32]byte)(&profile.UserIDAtServer),
		(*[32]byte)(&conf.TransportSecretKeyForServer),
		(*[32]byte)(&conf.ServerTransportPK))
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

	// load prekeys and ensure that we have enough of them
	prekeyPublics, prekeySecrets, err := LoadPrekeys(conf)
	if err != nil {
		return err
	}
	numKeys, err := util.GetNumKeys(ourConn, connToServer, conf.outBuf)
	if err != nil {
		return err
	}
	if numKeys < MIN_PREKEYS {
		newPublicPrekeys, newSecretPrekeys, err := GeneratePrekeys(MAX_PREKEYS - int(numKeys))
		prekeySecrets = append(prekeySecrets, newSecretPrekeys...)
		prekeyPublics = append(prekeyPublics, newPublicPrekeys...)
		if err = StorePrekeys(conf, prekeyPublics, prekeySecrets); err != nil {
			return err
		}
		var signingKey [64]byte
		copy(signingKey[:], conf.KeySigningSecretKey[:64])
		err = util.UploadKeys(ourConn, connToServer, conf.outBuf, util.SignKeys(newPublicPrekeys, &signingKey))
		if err != nil {
			return err // TODO handle this nicely
		}
	}

	initFn := func(path string, f os.FileInfo, err error) error {
		return processOutboxDir(conf, path)
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

	if err = util.EnablePush(ourConn, connToServer, conf.outBuf); err != nil {
		return err
	}

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

				processOutboxDir(conf, ev.Name)
			}
		case envelope := <-connToServer.ReadEnvelope:
			// assume it's the first message we're receiving from the person; try to decrypt
			message, ratch, index, err := conf.decryptFirstMessage(envelope, prekeyPublics, prekeySecrets)
			if err == nil {
				// assumption was correct, found a prekey that matched
				StoreRatchet(conf, message.Dename, ratch)

				//TODO: Update prekeys by removing index, store
				fmt.Printf("%s\n", message)
				index = index //Take out
				//TODO: Take out metadata + converastion from msg, Store the decrypted message
			} else { // try decrypting with a ratchet
				fillAuth := util.FillAuthWith((*[32]byte)(&conf.MessageAuthSecretKey))
				checkAuth := util.CheckAuthWith(conf.denameClient)
				ratchets, err := AllRatchets(conf, fillAuth, checkAuth)
				if err != nil {
					return err
				}
				fmt.Printf("Num ratchets: %d\n", len(ratchets))
				fmt.Printf("Ratchets: %#v\n", ratchets[0])

				message, ratch, err := conf.decryptMessage(envelope, ratchets)
				if err != nil {
					return err
				}
				fmt.Printf("%s\n", message)

				//TODO: Take out metadata + conversation from msg, then store
				StoreRatchet(conf, message.Dename, ratch)
			}
		case err := <-watcher.Error:
			if err != nil {
				return err
			}
		}
	}

}
