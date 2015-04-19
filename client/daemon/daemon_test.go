package daemon

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	util "github.com/andres-erbsen/chatterbox/client"
	"github.com/andres-erbsen/chatterbox/client/persistence"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/server"
	"github.com/andres-erbsen/chatterbox/shred"
	denameClient "github.com/andres-erbsen/dename/client"
	denameTestutil "github.com/andres-erbsen/dename/testutil"
	"gopkg.in/fsnotify.v1"
)

func TestReadFromFiles(t *testing.T) {
	alice := "alice"
	bob := "bob"

	denameConfig, denameTeardown := denameTestutil.SingleServer(t)
	defer denameTeardown()

	_, serverPubkey, serverAddr, serverTeardown := server.CreateTestServer(t)
	defer serverTeardown()

	aliceDir, err := ioutil.TempDir("", "daemon-alice")
	if err != nil {
		t.Fatal(err)
	}
	defer shred.RemoveAll(aliceDir)

	bobDir, err := ioutil.TempDir("", "daemon-bob")
	if err != nil {
		t.Fatal(err)
	}
	defer shred.RemoveAll(bobDir)

	aliceDaemon := PrepareTestAccountDaemon(alice, aliceDir, denameConfig, serverAddr, serverPubkey, t)
	bobDaemon := PrepareTestAccountDaemon(bob, bobDir, denameConfig, serverAddr, serverPubkey, t)

	//activate initialized daemons
	aliceDaemon.Start()
	bobDaemon.Start()
	defer aliceDaemon.Stop()
	defer bobDaemon.Stop()

	//alice creates a new conversation and sends a message
	participants := []string{alice, bob}
	subj := "testConversation"
	conv := &proto.ConversationMetadata{
		Participants: participants,
		Subject:      subj,
	}

	if err := aliceDaemon.ConversationToOutbox(conv); err != nil {
		t.Fatal(err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}

	err = watcher.Add(bobDaemon.ConversationDir())
	if err != nil {
		t.Fatal(err)
	}

	sentMessage := "Bob, can you hear me?"
	if err := aliceDaemon.MessageToOutbox(persistence.ConversationName(conv), sentMessage); err != nil {
		t.Fatal(err)
	}

	var files []string
	for len(files) == 0 {
		watcher.Add((<-watcher.Events).Name)
		if files, err = filepath.Glob(filepath.Join(bobDaemon.ConversationDir(), persistence.ConversationName(conv), "*alice")); err != nil {
			t.Fatal(err)
		}
	}

	receivedMessage, err := ioutil.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(receivedMessage) != sentMessage {
		t.Fatal("receivedMessage != sentMessage")
	}
}

func TestEncryptFirstMessage(t *testing.T) {
	alice := "alice"
	bob := "bob"

	denameConfig, denameTeardown := denameTestutil.SingleServer(t)
	defer denameTeardown()

	aliceDnmc, err := denameClient.NewClient(denameConfig, nil, nil)
	bobDnmc, err := denameClient.NewClient(denameConfig, nil, nil)

	if err != nil {
		t.Fatal(err)
	}

	_, serverPubkey, serverAddr, serverTeardown := server.CreateTestServer(t)
	defer serverTeardown()

	aliceDir, err := ioutil.TempDir("", "daemon-alice")
	if err != nil {
		t.Fatal(err)
	}
	defer shred.RemoveAll(aliceDir)
	bobDir, err := ioutil.TempDir("", "daemon-bob")
	if err != nil {
		t.Fatal(err)
	}
	defer shred.RemoveAll(bobDir)

	aliceConf := &Daemon{
		Paths: persistence.Paths{
			RootDir:     aliceDir,
			Application: "daemon",
		},
		Now:                  time.Now,
		foreignDenameClient:  aliceDnmc,
		timelessDenameClient: aliceDnmc,
		inBuf:                make([]byte, proto.SERVER_MESSAGE_SIZE),
		outBuf:               make([]byte, proto.SERVER_MESSAGE_SIZE),
		LocalAccountConfig:   proto.LocalAccountConfig{},
		LocalAccount: proto.LocalAccount{
			Dename: alice,
		},
		cc: util.NewConnectionCache(util.NewAnonDialer("DANGEROUS_NO_TOR")),
	}

	bobConf := &Daemon{
		Paths: persistence.Paths{
			RootDir:     bobDir,
			Application: "daemon",
		},
		Now:                  time.Now,
		foreignDenameClient:  bobDnmc,
		timelessDenameClient: bobDnmc,
		inBuf:                make([]byte, proto.SERVER_MESSAGE_SIZE),
		outBuf:               make([]byte, proto.SERVER_MESSAGE_SIZE),
		LocalAccountConfig:   proto.LocalAccountConfig{},
		LocalAccount: proto.LocalAccount{
			Dename: bob,
		},
		cc: util.NewConnectionCache(util.NewAnonDialer("DANGEROUS_NO_TOR")),
	}

	aliceHomeConn := util.CreateTestAccount(alice, aliceDnmc, &aliceConf.LocalAccountConfig, serverAddr, serverPubkey, t)
	defer aliceHomeConn.Close()
	bobHomeConn := util.CreateTestAccount(bob, bobDnmc, &bobConf.LocalAccountConfig, serverAddr, serverPubkey, t)
	defer bobHomeConn.Close()

	//fmt.Printf("CBob: %v\n", ([32]byte)(bobConf.TransportSecretKeyForServer))
	aliceNotifies := make(chan *util.EnvelopeWithId)
	aliceReplies := make(chan *proto.ServerToClient)

	aliceConnToServer := &util.ConnectionToServer{
		InBuf:        aliceConf.inBuf,
		Conn:         aliceHomeConn,
		ReadReply:    aliceReplies,
		ReadEnvelope: aliceNotifies,
	}

	go aliceConnToServer.ReceiveMessages()

	bobNotifies := make(chan *util.EnvelopeWithId)
	bobReplies := make(chan *proto.ServerToClient)

	bobConnToServer := &util.ConnectionToServer{
		InBuf:        bobConf.inBuf,
		Conn:         bobHomeConn,
		ReadReply:    bobReplies,
		ReadEnvelope: bobNotifies,
	}

	go bobConnToServer.ReceiveMessages()

	if err := InitFs(aliceConf); err != nil {
		t.Fatal(err)
	}

	if err := InitFs(bobConf); err != nil {
		t.Fatal(err)
	}
	//Bob uploads keys
	bobPublicPrekeys, bobSecretPrekeys, err := GeneratePrekeys(maxPrekeys)
	var bobSigningKey [64]byte
	copy(bobSigningKey[:], bobConf.KeySigningSecretKey[:64])
	err = util.UploadKeys(bobConnToServer, util.SignKeys(bobPublicPrekeys, &bobSigningKey))
	if err != nil {
		t.Fatal(err)
	}

	//Bob enables notifications
	if err = util.EnablePush(bobConnToServer); err != nil {
		t.Fatal(err)
	}

	//Alice uploads keys
	alicePublicPrekeys, _, err := GeneratePrekeys(maxPrekeys)
	var aliceSigningKey [64]byte
	copy(aliceSigningKey[:], aliceConf.KeySigningSecretKey[:64])
	err = util.UploadKeys(aliceConnToServer, util.SignKeys(alicePublicPrekeys, &aliceSigningKey))
	if err != nil {
		t.Fatal(err)
	}

	//Alice enables notification
	if err = util.EnablePush(aliceConnToServer); err != nil {
		t.Fatal(err)
	}

	participants := make([]string, 0, 2)
	participants = append(participants, alice)
	participants = append(participants, bob)

	msg1 := []byte("Envelope")

	payload := proto.Message{
		Subject:      "Subject1",
		Participants: participants,
		Dename:       alice,
		Contents:     msg1,
	}
	envelope, err := payload.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	err = aliceConf.sendFirstMessage(envelope, bob)
	if err != nil {
		t.Fatal(err)
	}
	incoming := <-bobConnToServer.ReadEnvelope

	out, bobRatch, _, err := bobConf.decryptFirstMessage(incoming.Envelope, bobPublicPrekeys, bobSecretPrekeys)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Bob hears: %s\n", out)

	msg2 := []byte("Envelope2")
	payload2 := proto.Message{
		Subject:      "Subject3",
		Participants: participants,
		Dename:       bob,
		Contents:     msg2,
	}
	envelope2, err := payload2.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	err = bobConf.sendMessage(envelope2, alice, bobRatch)
	if err != nil {
		t.Fatal(err)
	}

	incomingAlice := <-aliceConnToServer.ReadEnvelope

	aliceConf.fillAuth = util.FillAuthWith((*[32]byte)(&aliceConf.MessageAuthSecretKey))
	aliceConf.checkAuth = util.CheckAuthWith(aliceConf.ProfileRatchet)
	aliceRatchets, err := AllRatchets(aliceConf, aliceConf.fillAuth, aliceConf.checkAuth)
	outAlice, _, err := decryptMessage(incomingAlice.Envelope, aliceRatchets)
	if err != nil {
		t.Fatal(err)
	}

	ha := sha256.Sum256(incomingAlice.Envelope)
	if err := aliceConf.saveMessage(outAlice); err != nil {
		t.Fatal(err)
	}
	if err := util.DeleteMessages(aliceConnToServer, []*[32]byte{&ha}); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Alice hears: %s\n", outAlice)

	//TODO: Confirm message is as expected within the test
}
