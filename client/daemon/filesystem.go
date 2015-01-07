// file system utility

package daemon

import (
	"code.google.com/p/go.exp/fsnotify"
	"fmt"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/ratchet"
	"github.com/andres-erbsen/chatterbox/shred"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

func (conf *Config) ConfigFile() string {
	return filepath.Join(conf.RootDir, "config.pb")
}

func (conf *Config) ConversationDir() string {
	return filepath.Join(conf.RootDir, "conversations")
}

func (conf *Config) OutboxDir() string {
	return filepath.Join(conf.RootDir, "outbox")
}

func (conf *Config) TmpDir() string {
	return filepath.Join(conf.RootDir, "tmp")
}

func (conf *Config) JournalDir() string {
	return filepath.Join(conf.RootDir, "journal")
}

func (conf *Config) KeysDir() string {
	return filepath.Join(conf.RootDir, "keys")
}

func (conf *Config) RatchetKeysDir() string {
	return filepath.Join(conf.RootDir, "keys", "ratchet")
}

func (conf *Config) UiInfoDir() string {
	return filepath.Join(conf.RootDir, "ui_info")
}

func (conf *Config) UniqueTmpDir() (string, error) {
	return ioutil.TempDir(conf.TmpDir(), conf.TempPrefix)
}

const (
	MetadataFileName           = "metadata.pb"
	PrekeysFileName            = "prekeys.pb"
	LocalAccountConfigFileName = "config.pb"
	ProfileFileName            = "profile.pb"
)

func GenerateConversationName(sender string, metadata *proto.ConversationMetadata) string {
	//dirName := "date-sender-recipient-recipient"
	dateStr := time.Unix(0, metadata.Date).UTC().Format(time.RFC3339)
	recipientStrings := make([]string, 0, len(metadata.Participants))
	for i := 0; i < len(metadata.Participants); i++ {
		if metadata.Participants[i] != sender {
			recipientStrings = append(recipientStrings, string(metadata.Participants[i]))
		}
	}
	sort.Strings(recipientStrings)
	recipientsStr := strings.Join(recipientStrings, "-")
	dirName := fmt.Sprintf("%s-%s-%s", dateStr, sender, recipientsStr)
	return dirName
}

func GenerateMessageName(date time.Time, sender string) string {
	//messageName := "date-sender"
	dateStr := date.UTC().Format(time.RFC3339)
	return fmt.Sprintf("%s-%s", dateStr, sender)
}

func Copy(source string, dest string, perm os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)

	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)

	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

func UnmarshalFromFile(path string, out interface {
	Unmarshal([]byte) error
}) error {
	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return out.Unmarshal(fileContents)
}

func MarshalToFile(conf *Config, path string, in interface {
	Marshal() ([]byte, error)
}) error {
	inBytes, err := in.Marshal()
	if err != nil {
		return err
	}

	tmpDir, err := conf.UniqueTmpDir()
	if err != nil {
		return err
	}
	defer shred.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, filepath.Base(path))
	err = ioutil.WriteFile(tmpFile, inBytes, 0600)
	if err != nil {
		return err
	}

	err = os.Rename(path, tmpFile+".old")
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	err = os.Rename(tmpFile, path)
	if err != nil {
		return err
	}

	return nil
}

func LoadPrekeys(conf *Config) ([]*[32]byte, []*[32]byte, error) {
	prekeysProto := new(proto.Prekeys)
	err := UnmarshalFromFile(filepath.Join((*conf).KeysDir(), PrekeysFileName), prekeysProto)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	if len(prekeysProto.PrekeyPublics) != len(prekeysProto.PrekeySecrets) {
		return nil, nil, fmt.Errorf("len(prekeysProto.prekeyPublics) != len(prekeysProto.prekeySecrets)")
	}
	// convert protobuf proto.Byte32 to *[32]byte
	prekeySecrets := make([]*[32]byte, len(prekeysProto.PrekeySecrets))
	prekeyPublics := make([]*[32]byte, len(prekeysProto.PrekeyPublics))
	for i := 0; i < len(prekeySecrets); i++ {
		prekeySecrets[i] = (*[32]byte)(&prekeysProto.PrekeySecrets[i])
		prekeyPublics[i] = (*[32]byte)(&prekeysProto.PrekeyPublics[i])
	}
	return prekeyPublics, prekeySecrets, nil
}

func StorePrekeys(conf *Config, prekeyPublics, prekeySecrets []*[32]byte) error {
	if len(prekeyPublics) != len(prekeySecrets) {
		panic("len(prekeysPublics) != len(prekeySecrets)")
	}
	// convert [32]byte to proto.Byte32
	prekeysProto := proto.Prekeys{
		PrekeySecrets: make([]proto.Byte32, len(prekeySecrets)),
		PrekeyPublics: make([]proto.Byte32, len(prekeySecrets)),
	}
	for i := 0; i < len(prekeyPublics); i++ {
		prekeysProto.PrekeySecrets[i] = (proto.Byte32)(*prekeySecrets[i])
		prekeysProto.PrekeyPublics[i] = (proto.Byte32)(*prekeyPublics[i])
	}
	return MarshalToFile(conf, filepath.Join(conf.KeysDir(), PrekeysFileName), &prekeysProto)
}

func LoadLocalAccountConfig(conf *Config) (*proto.LocalAccountConfig, error) {
	localAccountProto := new(proto.LocalAccountConfig)
	return localAccountProto, UnmarshalFromFile(filepath.Join(conf.KeysDir(), LocalAccountConfigFileName), localAccountProto)
}

func StoreLocalAccountConfig(conf *Config, localAccountConfig *proto.LocalAccountConfig) error {
	return MarshalToFile(conf, filepath.Join(conf.KeysDir(), LocalAccountConfigFileName), localAccountConfig)
}

func LoadPublicProfile(conf *Config) (*proto.Profile, error) {
	profileProto := new(proto.Profile)
	return profileProto, UnmarshalFromFile(filepath.Join(conf.RootDir, ProfileFileName), profileProto)
}

func StorePublicProfile(conf *Config, publicProfile *proto.Profile) error {
	return MarshalToFile(conf, filepath.Join(conf.RootDir, ProfileFileName), publicProfile)
}

func LoadRatchet(conf *Config, name string, fillAuth func(tag, data []byte, theirAuthPublic *[32]byte), checkAuth func(tag, data, msg []byte, ourAuthPrivate *[32]byte) error) (*ratchet.Ratchet, error) {
	nameStr := string(name)
	// TODO: move name validation to the first place where we encoiunter a name
	if err := ValidateName(nameStr); err != nil {
		return nil, err
	}
	ratch := new(ratchet.Ratchet)
	if err := UnmarshalFromFile(filepath.Join(conf.RatchetKeysDir(), nameStr), ratch); err != nil {
		return nil, err
	}
	ratch.FillAuth = fillAuth
	ratch.CheckAuth = checkAuth
	return ratch, nil
}

func StoreRatchet(conf *Config, name string, ratch *ratchet.Ratchet) error {
	nameStr := string(name)

	// TODO: move name validation to the first place where we encoiunter a name
	if err := ValidateName(nameStr); err != nil {
		return err
	}
	return MarshalToFile(conf, filepath.Join(conf.RatchetKeysDir(), nameStr), ratch)
}

func AllRatchets(conf *Config, fillAuth func(tag, data []byte, theirAuthPublic *[32]byte), checkAuth func(tag, data, msg []byte, ourAuthPrivate *[32]byte) error) ([]*ratchet.Ratchet, error) {
	files, err := ioutil.ReadDir(conf.RatchetKeysDir())
	if err != nil {
		return nil, err
	}
	ret := make([]*ratchet.Ratchet, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ratch, err := LoadRatchet(conf, file.Name(), fillAuth, checkAuth)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ratchet for \"%s\": %s", file.Name(), err)
		}
		ret = append(ret, ratch)
	}
	return ret, nil
}

func InitFs(conf *Config) error {
	// create root directory and immediate sub directories
	os.MkdirAll(conf.RootDir, 0700)
	subdirs := []string{
		conf.ConversationDir(),
		conf.OutboxDir(),
		conf.TmpDir(),
		conf.JournalDir(),
		conf.KeysDir(),
		conf.RatchetKeysDir(),
		conf.UiInfoDir(),
	}
	for _, dir := range subdirs {
		os.Mkdir(dir, 0700)
	}

	// for each existing conversation, create a folder in the outbox
	copyToOutbox := func(cPath string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if cPath != conf.ConversationDir() {
			if f.IsDir() {
				// create the outbox directory in tmp, then (atomically) move it to outbox
				tmpDir, err := conf.UniqueTmpDir()
				if err != nil {
					return err
				}
				defer shred.RemoveAll(tmpDir)
				conversationInfo, err := os.Stat(cPath)
				if err != nil {
					return err
				}
				var c_perm = conversationInfo.Mode()
				metadataFile := filepath.Join(cPath, MetadataFileName)
				metadataInfo, err := os.Stat(metadataFile)
				if err != nil {
					return err
				}
				var m_perm = metadataInfo.Mode()
				oldUmask := syscall.Umask(0000)
				defer syscall.Umask(oldUmask)
				os.Mkdir(filepath.Join(tmpDir, path.Base(cPath)), c_perm)
				err = Copy(metadataFile, filepath.Join(tmpDir, path.Base(cPath), MetadataFileName), m_perm)
				if err != nil {
					return err
				}
				err = os.Rename(filepath.Join(tmpDir, path.Base(cPath)), filepath.Join(conf.OutboxDir(), path.Base(cPath)))
				if err != nil {
					// skip this conversation; this probably means it already exists in the outbox
					return nil
				}
			}
		}
		return nil
	}
	err := filepath.Walk(conf.ConversationDir(), copyToOutbox)
	if err != nil {
		return err
	}
	return nil
}

func WatchDir(watcher *fsnotify.Watcher, dir string, initFn filepath.WalkFunc) error {
	registerAndInit := func(path string, f os.FileInfo, err error) error {
		err = watcher.Watch(path)
		if err != nil {
			return err
		}
		err = initFn(path, f, err)
		if err != nil {
			return err
		}
		return nil
	}

	err := filepath.Walk(dir, registerAndInit)
	if err != nil {
		return err
	}

	return nil
}
