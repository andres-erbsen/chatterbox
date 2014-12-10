// file system utility

package daemon

import (
	"code.google.com/p/go.exp/fsnotify"
	"github.com/andres-erbsen/chatterbox/proto"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"syscall"
)

func GetConversationDir(conf Config) string {
	return filepath.Join(conf.RootDir, "conversations")
}

func GetOutboxDir(conf Config) string {
	return filepath.Join(conf.RootDir, "outbox")
}

func getTmpDir(conf Config) string {
	return filepath.Join(conf.RootDir, "tmp")
}

func getJournalDir(conf Config) string {
	return filepath.Join(conf.RootDir, "journal")
}

func getKeysDir(conf Config) string {
	return filepath.Join(conf.RootDir, "keys")
}

func getRatchetKeysDir(conf Config) string {
	return filepath.Join(conf.RootDir, "keys", "ratchet")
}

func GetUiInfoDir(conf Config) string {
	return filepath.Join(conf.RootDir, "ui_info")
}

func GetUniqueTmpDir(conf Config) (string, error) {
	return ioutil.TempDir(getTmpDir(conf), conf.TempPrefix)
}

const (
	MetadataFileName = "metadata.pb"
	PrekeysFileName  = "prekeys.pb"
)

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

func LoadPrekeys(conf Config) (*proto.Prekeys, error) {
	prekeysBytes, err := ioutil.ReadFile(filepath.Join(getKeysDir(conf), PrekeysFileName))
	if err != nil {
		return nil, err
	}

	prekeysProto := new(proto.Prekeys)
	err = prekeysProto.Unmarshal(prekeysBytes)
	if err != nil {
		return nil, err
	}

	return prekeysProto, nil
}

func StorePrekeys(conf Config, prekeys *proto.Prekeys) error {
	prekeysBytes, err := prekeys.Marshal()
	if err != nil {
		return err
	}

	tmpDir, err := GetUniqueTmpDir(conf)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, PrekeysFileName)
	err = ioutil.WriteFile(tmpFile, prekeysBytes, 0600)
	if err != nil {
		return err
	}

	prekeysFile := filepath.Join(getKeysDir(conf), PrekeysFileName)
	err = os.Rename(prekeysFile, tmpFile+".old")
	if err != nil {
		return err
	}
	err = os.Rename(tmpFile, prekeysFile)
	if err != nil {
		return err
	}

	return nil
}

func InitFs(conf Config) error {
	// create root directory and immediate sub directories
	os.MkdirAll(conf.RootDir, 0700)
	subdirs := []string{
		GetConversationDir(conf),
		GetOutboxDir(conf),
		getTmpDir(conf),
		getJournalDir(conf),
		getKeysDir(conf),
		getRatchetKeysDir(conf),
		GetUiInfoDir(conf),
	}
	for _, dir := range subdirs {
		os.Mkdir(dir, 0700)
	}

	// for each existing conversation, create a folder in the outbox
	copyToOutbox := func(cPath string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if cPath != GetConversationDir(conf) {
			if f.IsDir() {
				// create the outbox directory in tmp, then (atomically) move it to outbox
				tmpDir, err := GetUniqueTmpDir(conf)
				if err != nil {
					return err
				}
				defer os.RemoveAll(tmpDir)
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
				err = os.Rename(filepath.Join(tmpDir, path.Base(cPath)), filepath.Join(GetOutboxDir(conf), path.Base(cPath)))
				if err != nil {
					// skip this conversation; this probably means it already exists in the outbox
					return nil
				}
			}
		}
		return nil
	}
	err := filepath.Walk(GetConversationDir(conf), copyToOutbox)
	if err != nil {
		return err
	}
	return nil
}

func WatchDir(watcher *fsnotify.Watcher, dir string, initFn filepath.WalkFunc) error {
	registerAndInit := func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			err = watcher.Watch(path)
			if err != nil {
				return err
			}
			err = initFn(path, f, err)
			if err != nil {
				return err
			}
		}
		return nil
	}

	err := filepath.Walk(dir, registerAndInit)
	if err != nil {
		return err
	}

	return nil
}
