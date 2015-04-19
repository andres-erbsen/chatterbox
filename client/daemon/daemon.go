//   watches the file system for new messages --> sends them
//   communicates with the server --> receive new messages
package daemon

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"crypto/rand"
	"crypto/sha256"

	"code.google.com/p/go.exp/fsnotify"
	util "github.com/andres-erbsen/chatterbox/client"
	"github.com/andres-erbsen/chatterbox/client/persistence"
	"github.com/andres-erbsen/chatterbox/client/profilesyncd"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/chatterbox/shred"
	"github.com/andres-erbsen/dename/client"
	dename "github.com/andres-erbsen/dename/protocol"
)

const (
	// How many prekeys should the daemon try to keep at the server?
	maxPrekeys  = 100
	minPrekeys  = 50
	daemonAppID = "daemon"
)

// Daemon encapsulates long-running client-side chatterbox functionality
type Daemon struct {
	persistence.Paths

	// Gets the current time
	Now func() time.Time

	proto.LocalAccountConfig
	proto.LocalAccount

	foreignDenameClient  *client.Client
	timelessDenameClient *client.Client

	inBuf  []byte
	outBuf []byte

	stop chan struct{}
	wg   sync.WaitGroup
	psd  *profilesyncd.ProfileSyncd

	ourDenameLookup   *dename.ClientReply
	ourDenameLookupMu sync.Mutex

	checkAuth func(tag, data, msg []byte, ourAuthPrivate *[32]byte) error
	fillAuth  func(tag, data []byte, theirAuthPublic *[32]byte)

	cc *util.ConnectionCache
}

// Init creates a new account locally and at the server
func Init(rootDir, dename, serverAddr string, serverPort int, serverPK *[32]byte, torAddr string) error {
	d := &Daemon{
		Paths: persistence.Paths{
			RootDir:     rootDir,
			Application: "daemon",
		},
		LocalAccount: proto.LocalAccount{
			Dename: dename,
		},
		LocalAccountConfig: proto.LocalAccountConfig{
			ServerAddressTCP:  serverAddr,
			ServerPortTCP:     int32(serverPort),
			ServerTransportPK: (proto.Byte32)(*serverPK),
			TorAddress:        torAddr,
		},
		Now: time.Now,
		cc:  util.NewConnectionCache(util.NewAnonDialer(torAddr)),

		inBuf:  make([]byte, proto.SERVER_MESSAGE_SIZE),
		outBuf: make([]byte, proto.SERVER_MESSAGE_SIZE),
	}

	publicProfile := &proto.Profile{
		ServerAddressTCP:  serverAddr,
		ServerPortTCP:     int32(serverPort),
		ServerTransportPK: (proto.Byte32)(*serverPK),
	}

	if err := util.GenerateLongTermKeys(&d.LocalAccountConfig, publicProfile, rand.Reader); err != nil {
		panic(err)
	}

	conn, err := d.cc.DialServer(dename, serverAddr, serverPort, serverPK,
		(*[32]byte)(&publicProfile.UserIDAtServer), (*[32]byte)(&d.TransportSecretKeyForServer))
	if err != nil {
		return err
	}

	if err := os.MkdirAll(rootDir, 0700); err != nil {
		return err
	}
	if err := os.Mkdir(d.privDir(), 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(d.TempDir(), 0700); err != nil {
		return err
	}
	if err := d.MarshalToFile(d.OurChatterboxProfilePath(), publicProfile); err != nil {
		return err
	}
	if err := d.MarshalToFile(d.AccountPath(), &d.LocalAccount); err != nil {
		return err
	}
	if err := d.MarshalToFile(d.configPath(), &d.LocalAccountConfig); err != nil {
		return err
	}
	if err := d.MarshalToFile(d.OurChatterboxProfilePath(), publicProfile); err != nil {
		return err
	}
	defer d.cc.PutClose(dename)
	defer conn.Close()

	err = util.CreateAccount(conn, make([]byte, proto.SERVER_MESSAGE_SIZE))
	if err != nil {
		return err
	}

	notifies := make(chan *util.EnvelopeWithId)
	replies := make(chan *proto.ServerToClient)

	connToServer := &util.ConnectionToServer{
		InBuf:        d.inBuf,
		Conn:         conn,
		ReadReply:    replies,
		ReadEnvelope: notifies,
		Shutdown:     make(chan struct{}),
	}

	connToServer.WaitShutdown.Add(1)
	go func() { connToServer.ReceiveMessages(); connToServer.WaitShutdown.Done() }()

	_, _, err = d.updatePrekeys(connToServer)
	if err != nil {
		return err
	}

	close(connToServer.Shutdown)
	connToServer.WaitShutdown.Wait()

	return nil
}

// Load initializes a chatterbox daemon from rootDir
func Load(rootDir string, denameConfig *client.Config) (*Daemon, error) {
	d := &Daemon{
		Paths: persistence.Paths{
			RootDir:     rootDir,
			Application: "daemon",
		},
		Now:    time.Now,
		inBuf:  make([]byte, proto.SERVER_MESSAGE_SIZE),
		outBuf: make([]byte, proto.SERVER_MESSAGE_SIZE),
	}

	if err := persistence.UnmarshalFromFile(d.configPath(), &d.LocalAccountConfig); err != nil {
		return nil, err
	}
	d.cc = util.NewConnectionCache(util.NewAnonDialer(d.TorAddress))

	if err := persistence.UnmarshalFromFile(d.AccountPath(), &d.LocalAccount); err != nil {
		return nil, err
	}

	if err := persistence.UnmarshalFromFile(d.configPath(), &d.LocalAccountConfig); err != nil {
		return nil, err
	}

	d.ourDenameLookup = new(dename.ClientReply)
	persistence.UnmarshalFromFile(d.ourDenameLookupReplyPath(), d.ourDenameLookup)

	// ensure that we have a correct directory structure
	// including a correctly-populated outbox
	if err := InitFs(d); err != nil {
		return nil, err
	}

	ourDenameClient, err := client.NewClient(denameConfig, util.NewAnonDialer(d.TorAddress), nil)
	if err != nil {
		return nil, err
	}
	d.foreignDenameClient, err = client.NewClient(denameConfig, util.NewAnonDialer(d.TorAddress), nil)
	if err != nil {
		return nil, err
	}
	if denameConfig == nil {
		denameConfig = &client.DefaultConfig
	}
	timelessCfg := *denameConfig // TODO: make very sure this is a deep copy
	timelessCfg.Freshness.Threshold = fmt.Sprintf("%dh", 100*365*24)
	d.timelessDenameClient, err = client.NewClient(&timelessCfg, nil, nil)
	if err != nil {
		return nil, err
	}

	d.psd, err = profilesyncd.New(ourDenameClient, 10*time.Minute, d.Dename, d.onOurDenameProfileDownload, nil)
	if err != nil {
		return nil, err
	}

	d.fillAuth = util.FillAuthWith((*[32]byte)(&d.MessageAuthSecretKey))
	d.checkAuth = util.CheckAuthWith(d.ProfileRatchet)

	return d, nil
}

// Start activates the already initialized chatterbox daemon
func (d *Daemon) Start() {
	d.stop = make(chan struct{})
	d.psd.Start()
	if d.ourDenameLookup == nil {
		d.psd.Force()
	}
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		err := d.run()
		if err != nil {
			log.Fatal(err)
		}
	}()
}

// Stop stops the daemon and returns when it has completely shut down
func (d *Daemon) Stop() {
	close(d.stop)
	d.psd.Stop()
	d.wg.Wait()
}

// run executes the main loop of the chatterbox daemon
func (d *Daemon) run() error {
	profile := new(proto.Profile)
	if err := persistence.UnmarshalFromFile(d.OurChatterboxProfilePath(), profile); err != nil {
		return err
	}

	ourConn, err := d.cc.DialServer(d.Dename, d.ServerAddressTCP, int(d.ServerPortTCP),
		(*[32]byte)(&d.ServerTransportPK), (*[32]byte)(&profile.UserIDAtServer),
		(*[32]byte)(&d.TransportSecretKeyForServer))
	if err != nil {
		return err
	}
	defer ourConn.Close()

	notifies := make(chan *util.EnvelopeWithId)
	replies := make(chan *proto.ServerToClient)

	connToServer := &util.ConnectionToServer{
		InBuf:        d.inBuf,
		Conn:         ourConn,
		ReadReply:    replies,
		ReadEnvelope: notifies,
	}

	go connToServer.ReceiveMessages()

	prekeyPublics, prekeySecrets, err := d.updatePrekeys(connToServer)
	if err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	initFn := func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return d.processOutboxDir(path)
		}
		return d.processOutboxDir(filepath.Dir(path))
	}

	err = WatchDir(watcher, d.OutboxDir(), initFn)
	if err != nil {
		return err
	}

	if err = util.EnablePush(connToServer); err != nil {
		return err
	}

	d.requestAllMessages(connToServer)

	for {
		select {
		case <-d.stop:
			return nil
		case ev := <-watcher.Event:
			// event in the directory structure; watch any new directories
			if _, err = os.Stat(ev.Name); err == nil {
				err = WatchDir(watcher, ev.Name, initFn)
				if err != nil {
					log.Printf("watch %s: %s", ev.Name, err) // TODO
				}
				d.processOutboxDir(ev.Name)
			}
		case envelopewithid := <-connToServer.ReadEnvelope:
			envelope := envelopewithid.Envelope
			id := envelopewithid.Id
			msgHash := sha256.Sum256(envelope)
			// assume it's the first message we're receiving from the person; try to decrypt
			message, ratch, index, err := d.decryptFirstMessage(envelope, prekeyPublics, prekeySecrets)
			if err == nil {
				// assumption was correct, found a prekey that matched
				if err := StoreRatchet(d, message.Dename, ratch); err != nil {
					return err
				}

				newPrekeyPublics := append(prekeyPublics[:index], prekeyPublics[index+1:]...)
				newPrekeySecrets := append(prekeySecrets[:index], prekeySecrets[index+1:]...)
				if err = StorePrekeys(d, newPrekeyPublics, newPrekeySecrets); err != nil {
					return err
				}
				//TODO: Update prekeys by removing index, store // is this done? What does it mean?
				if err := d.saveMessage(message); err != nil {
					return err
				}
				if err := util.DeleteMessages(connToServer, []*[32]byte{id}); err != nil {
					return err
				}
			} else { // try decrypting with a ratchet
				ratchets, err := AllRatchets(d, d.fillAuth, d.checkAuth)
				if err != nil {
					return err
				}

				// TODO: figure out what here should be atomic and comment
				if message, ratch, err := decryptMessage(envelope, ratchets); err == nil {
					if err := d.saveMessage(message); err != nil {
						return err
					}
					if err := StoreRatchet(d, message.Dename, ratch); err != nil {
						return err
					}
				} else {
					log.Printf("failed to decrypt %x: %s", msgHash, err)
				}
				if err := util.DeleteMessages(connToServer, []*[32]byte{id}); err != nil {
					return err
				}
			}
		case err := <-watcher.Error:
			if err != nil {
				return err
			}
		}
	}

}

func (d *Daemon) updatePrekeys(connToServer *util.ConnectionToServer) (prekeyPublics, prekeySecrets []*[32]byte, err error) {
	// load prekeys and ensure that we have enough of them

	prekeyPublics, prekeySecrets, err = LoadPrekeys(d)
	if err != nil {
		return nil, nil, err
	}
	numKeys, err := util.GetNumKeys(connToServer)
	if err != nil {
		return nil, nil, err
	}

	if numKeys < minPrekeys {
		newPublicPrekeys, newSecretPrekeys, err := GeneratePrekeys(maxPrekeys - int(numKeys))
		prekeySecrets = append(prekeySecrets, newSecretPrekeys...)
		prekeyPublics = append(prekeyPublics, newPublicPrekeys...)
		if err = StorePrekeys(d, prekeyPublics, prekeySecrets); err != nil {
			return nil, nil, err
		}
		var signingKey [64]byte
		copy(signingKey[:], d.KeySigningSecretKey[:64])
		err = util.UploadKeys(connToServer, util.SignKeys(newPublicPrekeys, &signingKey))
		if err != nil {
			return nil, nil, err // TODO handle this nicely
		}
	}
	err = nil
	return
}

func (d *Daemon) requestAllMessages(conn *util.ConnectionToServer) error {
	msgs, err := util.ListUserMessages(conn)
	if err != nil {
		return err
	}
	for _, msgHash := range msgs {
		if err := util.RequestMessage(conn, msgHash); err != nil {
			return err
		}
	}
	return nil
}

func (d *Daemon) ProfileRatchet(name string, reply *dename.ClientReply) (*dename.Profile, error) {
	if reply != nil {
		if profile, err := d.foreignDenameClient.LookupFromReply(name, reply); err == nil {
			// case 1: a fresh lookup is provided by the sender: remember and use it
			return d.LatestProfile(name, profile)
		}
	}
	stored, err := d.LatestProfile(name, nil)
	if err == nil && stored != nil {
		// case 2: if we already have a profile, we don't care about absolute freshness.
		// This is okay assuming that 1) the original profile we got was fresh at
		// some point and 2) any changes after that would be pushed to us by the
		// profile owner sing a case 1 message. We still ignore received profiles
		// that are older than the one currently stored.
		if reply != nil {
			if profile, err := d.timelessDenameClient.LookupFromReply(name, reply); err == nil {
				return d.LatestProfile(name, profile)
			}
		}
		return stored, nil
	}

	// case 3: look up the profile ourselves and remember it.  This should only
	// happen if somebody sends us a message and we receive it when its bundled
	// lookup is no longer fresh.
	profile, err := d.foreignDenameClient.Lookup(name)
	if err != nil {
		return nil, err
	}
	return d.LatestProfile(name, profile)
}

func (d *Daemon) onOurDenameProfileDownload(p *dename.Profile, r *dename.ClientReply, e error) {
	if e != nil {
		log.Printf("dename lookup: %s", e)
		return
	}
	d.ourDenameLookupMu.Lock()
	d.ourDenameLookup = r
	d.ourDenameLookupMu.Unlock()
	if err := d.MarshalToFile(d.ourDenameLookupReplyPath(), r); err != nil {
		log.Print(err)
	}
}

func (d *Daemon) sendFirstMessage(msg []byte, theirDename string) error {
	profile, err := d.foreignDenameClient.Lookup(theirDename)
	if err != nil {
		return err
	}
	if profile == nil {
		fmt.Errorf("unkown dename on to line: " + theirDename)
	}
	if err := d.MarshalToFile(d.profilePath(theirDename), profile); err != nil {
		return err
	}

	chatProfileBytes, err := client.GetProfileField(profile, util.PROFILE_FIELD_ID)
	if err != nil {
		return err
	}

	chatProfile := new(proto.Profile)
	if err := chatProfile.Unmarshal(chatProfileBytes); err != nil {
		return err
	}

	addr := chatProfile.ServerAddressTCP
	pkSig := (*[32]byte)(&chatProfile.KeySigningKey)
	port := (int)(chatProfile.ServerPortTCP)
	pkTransport := (*[32]byte)(&chatProfile.ServerTransportPK)
	theirPk := (*[32]byte)(&chatProfile.UserIDAtServer)

	ourSkAuth := (*[32]byte)(&d.MessageAuthSecretKey)

	theirConn, err := d.cc.DialServer(theirDename, addr, port, pkTransport, nil, nil)
	if err != nil {
		return err
	}

	theirInBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)
	theirKey, err := util.GetKey(theirConn, theirInBuf, theirPk, theirDename, pkSig)
	if err != nil {
		theirConn.Close()
		d.cc.PutClose(theirDename)
		return err
	}
	encMsg, ratch, err := util.EncryptAuthFirst(msg, ourSkAuth, theirKey, d.ProfileRatchet)
	if err != nil {
		theirConn.Close()
		d.cc.PutClose(theirDename)
		return err
	}
	if err := StoreRatchet(d, theirDename, ratch); err != nil {
		theirConn.Close()
		d.cc.PutClose(theirDename)
		return err
	}
	err = util.UploadMessageToUser(theirConn, theirInBuf, theirPk, encMsg)
	if err != nil {
		theirConn.Close()
		d.cc.PutClose(theirDename)
		return err
	}
	d.cc.Put(theirDename, theirConn)
	return nil
}

func (d *Daemon) sendMessage(msg []byte, theirDename string, msgRatch *ratchet.Ratchet) error {
	profile := new(dename.Profile)
	err := persistence.UnmarshalFromFile(d.profilePath(theirDename), profile)
	if err != nil {
		return err
	}

	chatProfileBytes, err := client.GetProfileField(profile, util.PROFILE_FIELD_ID)
	if err != nil {
		return err
	}

	chatProfile := new(proto.Profile)
	if err := chatProfile.Unmarshal(chatProfileBytes); err != nil {
		return err
	}

	addr := chatProfile.ServerAddressTCP
	port := (int)(chatProfile.ServerPortTCP)
	pkTransport := (*[32]byte)(&chatProfile.ServerTransportPK)
	theirPk := (*[32]byte)(&chatProfile.UserIDAtServer)

	if err != nil {
		return err
	}

	theirInBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)

	encMsg, ratch, err := util.EncryptAuth(msg, msgRatch)
	if err != nil {
		return err
	}

	theirConn, err := d.cc.DialServer(theirDename, addr, port, pkTransport, nil, nil)
	if err != nil {
		return err
	}
	if err := StoreRatchet(d, theirDename, ratch); err != nil {
		theirConn.Close()
		d.cc.PutClose(theirDename)
		return err
	}
	err = util.UploadMessageToUser(theirConn, theirInBuf, theirPk, encMsg)
	if err != nil {
		theirConn.Close()
		d.cc.PutClose(theirDename)
		return err
	}
	d.cc.Put(theirDename, theirConn)
	return nil
}

func (d *Daemon) decryptFirstMessage(envelope []byte, pkList []*[32]byte, skList []*[32]byte) (*proto.Message, *ratchet.Ratchet, int, error) {
	skAuth := (*[32]byte)(&d.MessageAuthSecretKey)
	ratch, msg, index, err := util.DecryptAuthFirst(envelope, pkList, skList, skAuth, d.ProfileRatchet)

	if err != nil {
		return nil, nil, -1, err
	}
	message := new(proto.Message)
	if err := message.Unmarshal(msg); err != nil {
		return nil, nil, -1, err
	}

	return message, ratch, index, nil
}

func decryptMessage(envelope []byte, ratchets []*ratchet.Ratchet) (*proto.Message, *ratchet.Ratchet, error) {
	var ratch *ratchet.Ratchet
	var msg []byte
	var err error
	for _, msgRatch := range ratchets {
		ratch, msg, err = util.DecryptAuth(envelope, msgRatch)
		if err == nil {
			break // found the right ratchet
		}
	}
	if msg == nil {
		return nil, nil, fmt.Errorf("could not find suitable ratchet: %v", err)
	}
	message := new(proto.Message)
	if err := message.Unmarshal(msg); err != nil {
		return nil, nil, err
	}
	return message, ratch, nil
}

func undupStrings(ss []string) []string {
	ret := []string{}
	seen := make(map[string]struct{})
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			ret = append(ret, s)
			seen[s] = struct{}{}
		}
	}
	return ret
}

func (d *Daemon) processOutboxDir(dirname string) error {
	// TODO: refactor: separate message assembly and filesystem access?
	// parse metadata
	metadataFile := filepath.Join(dirname, persistence.MetadataFileName)
	if _, err := os.Stat(metadataFile); err != nil {
		return nil // no metadata --> not an outgoing message
	}

	metadata := proto.ConversationMetadata{}
	err := persistence.UnmarshalFromFile(metadataFile, &metadata)
	if err != nil {
		return err
	}

	metadata.Participants = append(metadata.Participants, d.Dename)
	undupStrings(metadata.Participants)
	sort.Strings(metadata.Participants)
	convName := persistence.ConversationName(&metadata)

	// load messages
	potentialMessages, err := ioutil.ReadDir(dirname)
	if err != nil {
		return err
	}
	messages := make([][]byte, 0, len(potentialMessages))
	for _, finfo := range potentialMessages {
		if !finfo.IsDir() && finfo.Name() != persistence.MetadataFileName {
			msg, err := ioutil.ReadFile(filepath.Join(dirname, finfo.Name()))
			if err != nil {
				return err
			}

			// make protobuf for message; append it
			d.ourDenameLookupMu.Lock()
			payload := proto.Message{
				Dename:       d.Dename,
				DenameLookup: d.ourDenameLookup,
				Contents:     msg,
				Subject:      metadata.Subject,
				Participants: metadata.Participants,
				Date:         finfo.ModTime().UnixNano(),
			}
			d.ourDenameLookupMu.Unlock()
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

	if err := d.conversationToConversations(&metadata); err != nil && !os.IsExist(err) && !strings.Contains(fmt.Sprint(err), "directory not empty") {
		log.Fatal(err)
	}

	for _, recipient := range metadata.Participants {
		if recipient == d.Dename {
			continue
		}
		for _, msg := range messages {
			if err != nil {
				return err
			}
			if msgRatch, err := LoadRatchet(d, recipient, d.fillAuth, d.checkAuth); err != nil { //First message to this recipien
				if err := d.sendFirstMessage(msg, recipient); err != nil {
					return err
				}
			} else {
				if err := d.sendMessage(msg, recipient, msgRatch); err != nil {
					return err
				}
			}
		}
	}

	// move the sent messages to the conversation folder
	for _, finfo := range potentialMessages {
		if !finfo.IsDir() && finfo.Name() != persistence.MetadataFileName {
			if err = os.Rename(filepath.Join(dirname, finfo.Name()), filepath.Join(d.ConversationDir(), persistence.ConversationName(&metadata), persistence.MessageName(finfo.ModTime(), string(d.Dename)))); err != nil {
				log.Fatal(err)
			}
		}
	}

	// canonicalize the outbox folder name
	if dirname != filepath.Join(d.OutboxDir(), convName) {
		if err := os.Rename(dirname, filepath.Join(d.OutboxDir(), convName)); err != nil {
			shred.RemoveAll(dirname)
		}
	}

	return nil
}

func (d *Daemon) saveMessage(message *proto.Message) error {
	// generate metadata file
	metadata := proto.ConversationMetadata{
		Participants: message.Participants,
		Subject:      message.Subject,
	}
	// generate conversation name
	convName := persistence.ConversationName(&metadata)
	messageName := persistence.MessageName(time.Unix(0, message.Date), string(message.Dename))

	// create conversation directory if it doesn't already exist
	convDir := filepath.Join(d.ConversationDir(), convName)
	outboxDir := filepath.Join(d.OutboxDir(), convName)
	_, err := os.Stat(convDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if err != nil && os.IsNotExist(err) {
		// new message in existing conversation
		if err := d.conversationToConversations(&metadata); err != nil {
			return err
		}
	}

	err = d.AtomicWriteFile(filepath.Join(convDir, messageName), message.Contents, 0600)
	if err != nil {
		return err
	}

	// to outbox
	tdir, err := d.MkdirInTemp()
	if err != nil {
		return err
	}
	defer shred.RemoveAll(tdir)
	err = d.MarshalToFile(filepath.Join(tdir, persistence.MetadataFileName), &metadata)
	if err != nil {
		return err
	}
	if err = os.Rename(filepath.Join(tdir), outboxDir); err != nil && !os.IsExist(err) && !strings.Contains(err.Error(), "directory not empty") {
		return err
	}
	return nil
}

func (p *Daemon) conversationToConversations(metadata *proto.ConversationMetadata) error {
	path := filepath.Join(p.ConversationDir(), persistence.ConversationName(metadata))
	tmpDir, err := p.MkdirInTemp()
	if err != nil {
		return err
	}
	defer shred.RemoveAll(tmpDir)
	if err := p.MarshalToFile(filepath.Join(tmpDir, persistence.MetadataFileName), metadata); err != nil {
		return err
	}
	return os.Rename(filepath.Join(tmpDir), path)
}
