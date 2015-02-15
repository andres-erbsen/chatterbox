package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/andres-erbsen/chatterbox/client/persistence"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/shred"
)

// SpawnConversationInOutbox Spawns a new conversation in a user's outbox
//
// conf = configuration structure
// subject = subject of the new conversation
// recipients = dename names of the recipients
// messages = list of messages (each is a byte array) to put in the outbox
func SpawnConversationInOutbox(conf *persistence.Paths, subject string, recipients []string, messages [][]byte) error {
	if err := os.Mkdir(conf.TempDir(), 0700); err != nil && !os.IsExist(err) {
		return err
	}
	// create temp directory or error
	tmpDir, err := conf.MkdirInTemp()
	defer shred.RemoveAll(tmpDir)
	if err != nil {
		return err
	}

	// generate metadata
	metadata := &proto.ConversationMetadata{
		Participants: recipients,
		Subject:      subject,
	}

	// create folder for conversation with the conversation name
	dirName := persistence.ConversationName(metadata)
	if err := os.Mkdir(filepath.Join(tmpDir, dirName), 0700); err != nil {
		return err
	}

	// write metadata file
	metadataFile := filepath.Join(tmpDir, dirName, persistence.MetadataFileName)
	metadataBytes, err := metadata.Marshal()
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(metadataFile, metadataBytes, 0600); err != nil {
		return err
	}

	// write messages to files in the folder (or error)
	for _, message := range messages {
		f, err := ioutil.TempFile(filepath.Join(tmpDir, dirName), "")
		if err != nil {
			return err
		}
		if err = ioutil.WriteFile(f.Name(), message, 0600); err != nil {
			return err
		}
	}

	// move folder to the outbox (or error)
	return os.Rename(filepath.Join(tmpDir, dirName), filepath.Join(conf.OutboxDir(), dirName))
}

var root = flag.String("root", "", "chatterbox root directory")
var message = flag.String("message", "", "to be sent to all participants")
var subject = flag.String("subject", "", "used to refer to the conversation")

func main() {
	flag.Parse()
	conf := &persistence.Paths{
		RootDir:     *root,
		Application: "chat-create",
	}
	if *root == "" || *subject == "" {
		flag.Usage()
		log.Fatal("no root or subject specified")
	}

	if err := SpawnConversationInOutbox(conf, *subject, flag.Args(), [][]byte{([]byte)(*message)}); err != nil {
		log.Fatal(err)
	}
}
