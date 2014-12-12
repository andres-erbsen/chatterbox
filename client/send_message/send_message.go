package main

import (
	"fmt"
	"github.com/andres-erbsen/chatterbox/client/daemon"
	"github.com/andres-erbsen/chatterbox/proto"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// Spawns a new conversation in a user's outbox
//
// conf = configuration structure
// subject = subject of the new conversation
// recipients = dename names of the recipients
// messages = list of messages (each is a byte array) to put in the outbox
func SpawnConversationInOutbox(conf *daemon.Config, subject string, recipients [][]byte, messages [][]byte) error {
	// create temp directory or error
	tmpDir, err := conf.UniqueTmpDir()
	defer os.RemoveAll(tmpDir)
	if err != nil {
		return err
	}

	// generate metadata
	recipients = append(recipients, conf.Dename)
	metadata := &proto.ConversationMetadata{
		Participants: recipients,
		Subject:      subject,
		Date:         conf.Now().UnixNano(),
	}

	// create folder for conversation with the conversation name (or error?)
	dirName := daemon.GenerateConversationName(conf.Dename, metadata)
	os.MkdirAll(filepath.Join(tmpDir, dirName), 0700)

	// write metadata file
	metadataFile := filepath.Join(tmpDir, dirName, daemon.MetadataFileName)
	metadataBytes, err := metadata.Marshal()
	if err != nil {
		return err
	}
	ioutil.WriteFile(metadataFile, metadataBytes, 0600)

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
	err = os.Rename(filepath.Join(tmpDir, dirName), filepath.Join(conf.OutboxDir(), dirName))
	if err != nil {
		return err
	}

	return nil
}

func main() {
	args := os.Args[1:]
	if len(args) < 4 {
		fmt.Println("arguments: <root_dir> <user_dename> <subject> <message>")
		os.Exit(1)
	}

	rootDir := args[0]
	recipient := []byte(args[1])
	subject := args[2]
	message := args[3]

	conf := daemon.LoadConfig(&daemon.Config{
		RootDir:    rootDir,
		Now:        time.Now,
		TempPrefix: "some_ui",
	})

	SpawnConversationInOutbox(conf, subject, [][]byte{recipient}, [][]byte{([]byte)(message)})
}
