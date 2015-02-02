package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/andres-erbsen/chatterbox/client/persistence"
	"github.com/andres-erbsen/chatterbox/proto"
)

func TestSpawnConversationInOutbox(t *testing.T) {
	// init the file system + configuration structure
	rootDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}

	conf := &persistence.Paths{
		RootDir:     rootDir,
		Application: "some_ui",
	}

	err = os.Mkdir(conf.OutboxDir(), 0700)
	if err != nil {
		t.Error(err)
	}
	err = os.MkdirAll(conf.TempDir(), 0700)
	if err != nil {
		t.Error(err)
	}

	subject := "test subject"
	recipients := []string{"recipient_dename_b", "recipient_dename_a"}
	messages := [][]byte{[]byte("message1"), []byte("message2")}
	err = SpawnConversationInOutbox(conf, subject, recipients, messages)
	if err != nil {
		t.Error(err)
	}

	// check that a conversation exists in the outbox with the correct name
	outboxDir := conf.OutboxDir()
	expectedName := "test subject %between recipient_dename_b %and recipient_dename_a"
	_, err = os.Stat(filepath.Join(outboxDir, expectedName))
	if err != nil {
		t.Error(err)
	}

	// check that it has a valid metadata file
	metadataBytes, err := ioutil.ReadFile(filepath.Join(outboxDir, expectedName, persistence.MetadataFileName))
	if err != nil {
		t.Error(err)
	}
	metadataProto := new(proto.ConversationMetadata)
	err = metadataProto.Unmarshal(metadataBytes)
	if err != nil {
		t.Error(err)
	}

	// check that it has all message files; for now assume they have the correct contents
	files, err := ioutil.ReadDir(filepath.Join(outboxDir, expectedName))
	if err != nil {
		t.Error(err)
	}
	if len(files) != 3 { // metadata file + 2 messages
		t.Error(fmt.Sprintf("Wrong number of files %d in outgoing conversation; should be 3", len(files)))
	}

	// check that the temp directory has been cleaned up
	files, err = ioutil.ReadDir(conf.TempDir())
	if err != nil {
		t.Error(err)
	}
	if len(files) > 0 {
		t.Error("tmp directory not cleaned up")
	}
}
