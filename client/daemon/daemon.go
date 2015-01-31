// client daemon
//   watches the file system for new messages --> sends them
//   communicates with the server --> receive new messages
package daemon

import (
	"code.google.com/p/go.exp/fsnotify"
	"errors"
	"fmt"
	util "github.com/andres-erbsen/chatterbox/client"
	//"github.com/andres-erbsen/chatterbox/client/profilesyncd"
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

func New(rootDir string) (*Daemon, error) {
	conf := LoadConfig(&Daemon{
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

func (d *Daemon) sendFirstMessage(msg []byte, theirDename string) (*ratchet.Ratchet, error) {
	//If using TOR, dename client is fresh TOR connection
	profile, err := d.denameClient.Lookup(theirDename)
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

	ourSkAuth := (*[32]byte)(&d.MessageAuthSecretKey)

	theirInBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)

	theirConn, err := util.CreateForeignServerConn(theirDename, d.denameClient, addr, port, pkTransport)
	if err != nil {
		return nil, err
	}
	defer theirConn.Close()

	theirKey, err := util.GetKey(theirConn, theirInBuf, d.outBuf, theirPk, theirDename, pkSig)
	if err != nil {
		return nil, err
	}
	encMsg, ratch, err := util.EncryptAuthFirst(msg, ourSkAuth, theirKey, d.denameClient)
	if err != nil {
		return nil, err
	}
	err = util.UploadMessageToUser(theirConn, theirInBuf, d.outBuf, theirPk, encMsg)
	if err != nil {
		return nil, err
	}
	return ratch, nil
}

func (d *Daemon) sendMessage(msg []byte, theirDename string, msgRatch *ratchet.Ratchet) (*ratchet.Ratchet, error) {
	//If using TOR, dename client is fresh TOR connection
	profile, err := d.denameClient.Lookup(theirDename)
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

	theirConn, err := util.CreateForeignServerConn(theirDename, d.denameClient, addr, port, pkTransport)
	if err != nil {
		return nil, err
	}
	defer theirConn.Close()

	encMsg, ratch, err := util.EncryptAuth(msg, msgRatch)
	if err != nil {
		return nil, err
	}
	err = util.UploadMessageToUser(theirConn, theirInBuf, d.outBuf, theirPk, encMsg)
	if err != nil {
		return nil, err
	}
	return ratch, nil
}

func (d *Daemon) decryptFirstMessage(envelope []byte, pkList []*[32]byte, skList []*[32]byte) (*proto.Message, *ratchet.Ratchet, int, error) {
	skAuth := (*[32]byte)(&d.MessageAuthSecretKey)
	ratch, msg, index, err := util.DecryptAuthFirst(envelope, pkList, skList, skAuth, d.denameClient)

	if err != nil {
		return nil, nil, -1, err
	}
	message := new(proto.Message)
	if err := message.Unmarshal(msg); err != nil {
		return nil, nil, -1, err
	}

	return message, ratch, index, nil
}

func (d *Daemon) decryptMessage(envelope []byte, ratchets []*ratchet.Ratchet) (*proto.Message, *ratchet.Ratchet, error) {
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

func (d *Daemon) processOutboxDir(dirname string) error {
	fmt.Printf("processing outbox dir: %s\n", dirname)
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

	// load messages
	potentialMessages, err := ioutil.ReadDir(dirname)
	if err != nil {
		return err
	}
	messages := make([][]byte, 0, len(potentialMessages))
	sendTime := d.Now().UTC()
	for _, finfo := range potentialMessages {
		if !finfo.IsDir() && finfo.Name() != MetadataFileName {
			msg, err := ioutil.ReadFile(filepath.Join(dirname, finfo.Name()))
			if err != nil {
				return err
			}

			// make protobuf for message; append it
			payload := proto.Message{
				Dename:        d.Dename,
				Contents:      msg,
				Subject:       metadata.Subject,
				Participants:  metadata.Participants,
				Date:          sendTime.UnixNano(),
				InitialSender: metadata.InitialSender,
				InitialDate:   metadata.Date,
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

	// ensure the conversation directory exists
	convName, err := filepath.Rel(d.OutboxDir(), dirname)
	if err != nil {
		return err
	}
	convPath := filepath.Join(d.ConversationDir(), convName)
	if os.Mkdir(convPath, 0700); err != nil && !os.IsExist(err) {
		return err
	}

	// copy the metadata file to the conversation directory if it isn't already there
	convMetadataFile := filepath.Join(convPath, MetadataFileName)
	if _, err = os.Stat(convMetadataFile); err != nil {
		if os.IsNotExist(err) {
			if err = MarshalToFile(d, convMetadataFile, &metadata); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	for _, recipient := range metadata.Participants {
		if recipient == d.Dename {
			continue
		}
		for _, msg := range messages {
			fillAuth := util.FillAuthWith((*[32]byte)(&d.MessageAuthSecretKey))
			checkAuth := util.CheckAuthWith(d.denameClient)
			if err != nil {
				return err
			}
			if msgRatch, err := LoadRatchet(d, recipient, fillAuth, checkAuth); err != nil { //First message in this conversation
				ratch, err := d.sendFirstMessage(msg, recipient)
				if err != nil {
					return err
				}
				StoreRatchet(d, recipient, ratch)
			} else { // Not-first message in this conversation
				ratch, err := d.sendMessage(msg, recipient, msgRatch)
				if err != nil {
					return err
				}
				StoreRatchet(d, recipient, ratch)
			}
		}
	}

	// move the sent messages to the conversation folder
	for _, finfo := range potentialMessages {
		if !finfo.IsDir() && finfo.Name() != MetadataFileName {
			messageName := GenerateMessageName(sendTime, string(d.Dename))
			if err = os.Rename(filepath.Join(dirname, finfo.Name()), filepath.Join(convPath, messageName)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *Daemon) receiveMessage(message *proto.Message) error {
	fmt.Printf("%s\n", message)

	// generate metadata file
	metadata := proto.ConversationMetadata{
		Participants:  message.Participants,
		Subject:       message.Subject,
		Date:          message.InitialDate,
		InitialSender: message.InitialSender,
	}

	// generate conversation name
	convName := GenerateConversationName(message.InitialSender, &metadata)

	// create conversation directory if it doesn't already exist
	convDir := filepath.Join(d.ConversationDir(), convName)
	if err := os.Mkdir(convDir, 0700); err != nil && !os.IsExist(err) {
		return err
	}

	// create outbox directory if it doesn't already exist
	outDir := filepath.Join(d.OutboxDir(), convName)
	if err := os.Mkdir(outDir, 0700); err != nil && !os.IsExist(err) {
		return err
	}

	// create conversation metadata file if it doesn't already exist
	convMetadataFile := filepath.Join(convDir, MetadataFileName)
	if _, err := os.Stat(convMetadataFile); err != nil {
		if os.IsNotExist(err) {
			MarshalToFile(d, convMetadataFile, &metadata)
		} else {
			return err
		}
	}

	// create outbox metadata file if it doesn't already exist
	outMetadataFile := filepath.Join(outDir, MetadataFileName)
	if _, err := os.Stat(outMetadataFile); err != nil {
		if os.IsNotExist(err) {
			MarshalToFile(d, outMetadataFile, &metadata)
		} else {
			return err
		}
	}

	// generate the message name: date-sender
	messageName := GenerateMessageName(time.Unix(0, message.Date), string(message.Dename))
	fmt.Printf("new message name: %s\n", messageName)

	// write the message to the conversation folder
	if err := ioutil.WriteFile(filepath.Join(convDir, messageName), message.Contents, 0600); err != nil {
		return err
	}

	return nil
}

func (d *Daemon) Run(shutdown <-chan struct{}) error {
	profile, err := LoadPublicProfile(d)
	if err != nil {
		return err
	}

	ourConn, err := util.CreateHomeServerConn(
		d.ServerAddressTCP, (*[32]byte)(&profile.UserIDAtServer),
		(*[32]byte)(&d.TransportSecretKeyForServer),
		(*[32]byte)(&d.ServerTransportPK))
	if err != nil {
		return err
	}
	defer ourConn.Close()

	notifies := make(chan []byte)
	replies := make(chan *proto.ServerToClient)

	connToServer := &util.ConnectionToServer{
		Buf:          d.inBuf,
		Conn:         ourConn,
		ReadReply:    replies,
		ReadEnvelope: notifies,
	}

	go connToServer.ReceiveMessages()

	// load prekeys and ensure that we have enough of them
	prekeyPublics, prekeySecrets, err := LoadPrekeys(d)
	if err != nil {
		return err
	}
	numKeys, err := util.GetNumKeys(ourConn, connToServer, d.outBuf)
	if err != nil {
		return err
	}
	if numKeys < MIN_PREKEYS {
		newPublicPrekeys, newSecretPrekeys, err := GeneratePrekeys(MAX_PREKEYS - int(numKeys))
		prekeySecrets = append(prekeySecrets, newSecretPrekeys...)
		prekeyPublics = append(prekeyPublics, newPublicPrekeys...)
		if err = StorePrekeys(d, prekeyPublics, prekeySecrets); err != nil {
			return err
		}
		var signingKey [64]byte
		copy(signingKey[:], d.KeySigningSecretKey[:64])
		err = util.UploadKeys(ourConn, connToServer, d.outBuf, util.SignKeys(newPublicPrekeys, &signingKey))
		if err != nil {
			return err // TODO handle this nicely
		}
	}

	initFn := func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return d.processOutboxDir(path)
		} else {
			return d.processOutboxDir(filepath.Dir(path))
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = WatchDir(watcher, d.OutboxDir(), initFn)
	if err != nil {
		return err
	}

	if err = util.EnablePush(ourConn, connToServer, d.outBuf); err != nil {
		return err
	}

	for {
		select {
		case <-shutdown:
			return nil
		case ev := <-watcher.Event:
			fmt.Printf("event: %v\n", ev)
			// event in the directory structure; watch any new directories
			if _, err = os.Stat(ev.Name); err == nil {
				err = WatchDir(watcher, ev.Name, initFn)
				if err != nil {
					return err
				}

				d.processOutboxDir(ev.Name)
			}
		case envelope := <-connToServer.ReadEnvelope:
			// assume it's the first message we're receiving from the person; try to decrypt
			message, ratch, index, err := d.decryptFirstMessage(envelope, prekeyPublics, prekeySecrets)
			if err == nil {
				// assumption was correct, found a prekey that matched
				StoreRatchet(d, message.Dename, ratch)

				//TODO: Update prekeys by removing index, store
				if err = d.receiveMessage(message); err != nil {
					return err
				}
				newPrekeyPublics := append(prekeyPublics[:index], prekeyPublics[index+1:]...)
				newPrekeySecrets := append(prekeySecrets[:index], prekeySecrets[index+1:]...)
				if err = StorePrekeys(d, newPrekeyPublics, newPrekeySecrets); err != nil {
					return err
				}
			} else { // try decrypting with a ratchet
				fillAuth := util.FillAuthWith((*[32]byte)(&d.MessageAuthSecretKey))
				checkAuth := util.CheckAuthWith(d.denameClient)
				ratchets, err := AllRatchets(d, fillAuth, checkAuth)
				if err != nil {
					return err
				}

				message, ratch, err := d.decryptMessage(envelope, ratchets)
				if err != nil {
					return err
				}
				if err = d.receiveMessage(message); err != nil {
					return err
				}

				StoreRatchet(d, message.Dename, ratch)
			}
		case err := <-watcher.Error:
			if err != nil {
				return err
			}
		}
	}

}
