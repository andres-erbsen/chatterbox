package persistence

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/andres-erbsen/chatterbox/client/encoding"
	"github.com/andres-erbsen/chatterbox/proto"
)

type Paths struct {
	RootDir     string
	Application string
}

const (
	MetadataFileName = "metadata.pb"
)

func (p *Paths) ConversationDir() string { return filepath.Join(p.RootDir, "conversations") }

func (p *Paths) OutboxDir() string { return filepath.Join(p.RootDir, "outbox") }

func (p *Paths) TempDir() string {
	return filepath.Join(p.RootDir, ".tmp", p.Application)
}

func ConversationName(metadata *proto.ConversationMetadata) string {
	names := make([]string, 0, len(metadata.Participants))
	already := make(map[string]struct{})
	for _, s := range metadata.Participants {
		if _, ok := already[s]; !ok {
			names = append(names, encoding.EscapeFilename(s))
			already[s] = struct{}{}
		}
	}
	sort.Strings(names)
	return encoding.EscapeFilename(metadata.Subject) + " %between " + strings.Join(names, " %and ")
}

func MessageName(date time.Time, sender string) string {
	//messageName := "date-sender"
	dateStr := date.UTC().Format(time.RFC3339)
	return fmt.Sprintf("%s-%s", dateStr, sender)
}

func (p *Paths) MkdirInTemp() (string, error) {
	return ioutil.TempDir(p.TempDir(), "")
}

// Unmarshal reads an Unmarshal()-able from a file
func UnmarshalFromFile(path string, out interface {
	Unmarshal([]byte) error
}) error {
	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return out.Unmarshal(fileContents)
}

// MarshalToFile atomically writes a Marshal()-able object to a file by first
// writing to a new file in tmpDir and then atomically renaming it to the
// destination file.
func (p *Paths) MarshalToFile(path string, in interface {
	Marshal() ([]byte, error)
}) error {
	inBytes, err := in.Marshal()
	if err != nil {
		return err
	}
	tmpFile := filepath.Join(p.TempDir(), filepath.Base(path)+"-"+randHex(16))
	err = ioutil.WriteFile(tmpFile, inBytes, 0600)
	if err != nil {
		return err
	}
	err = os.Rename(tmpFile, path)
	if err != nil {
		return err
	}
	return nil
}

func (p *Paths) ConversationToOutbox(metadata *proto.ConversationMetadata) error {
	return os.Mkdir(filepath.Join(p.OutboxDir(), ConversationName(metadata)), 0700)
}

func (p *Paths) MessageToOutbox(conversationName, message string) error {
	f, err := ioutil.TempFile(p.TempDir(), "")
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(f.Name(), []byte(message), 0600); err != nil {
		return err
	}

	return os.Rename(filepath.Join(p.TempDir(), f.Name()), filepath.Join(p.OutboxDir(), f.Name()))
}

func randHex(l int) string {
	s := make([]byte, (l+1)/2)
	if _, err := rand.Read(s); err != nil {
		panic(err)
	}
	return hex.EncodeToString(s)[:l]
}
