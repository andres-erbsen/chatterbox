// file system utility directory

package filesystem

import (
	"code.google.com/p/go.exp/fsnotify"
	"github.com/andres-erbsen/chatterbox/client/util/config"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"syscall"
)

func GetConversationDir(conf config.Config) string {
	return filepath.Join(conf.RootDir, "conversations")
}

func GetOutboxDir(conf config.Config) string {
	return filepath.Join(conf.RootDir, "outbox")
}

func GetTmpDir(conf config.Config) string {
	return filepath.Join(conf.RootDir, "tmp")
}

func getJournalDir(conf config.Config) string {
	return filepath.Join(conf.RootDir, "journal")
}

func GetKeysDir(conf config.Config) string {
	return filepath.Join(conf.RootDir, "keys")
}

func GetUiInfoDir(conf config.Config) string {
	return filepath.Join(conf.RootDir, "ui_info")
}

func GetUniqueTmpDir(conf config.Config) (string, error) {
	return ioutil.TempDir(GetTmpDir(conf), conf.TempPrefix)
}

const (
	MetadataFileName = "METADATA"
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
func InitFs(conf config.Config) error {
	// create root directory and immediate sub directories
	os.MkdirAll(conf.RootDir, 0700)
	subdirs := []string{
		GetConversationDir(conf),
		GetOutboxDir(conf),
		GetTmpDir(conf),
		getJournalDir(conf),
		GetKeysDir(conf),
		GetUiInfoDir(conf),
	}
	for _, dir := range subdirs {
		os.Mkdir(dir, 0700)
	}

	// for each existing conversation, create a folder in the outbox
	copyToOutbox := func(cPath string, f os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error reading conversation directories %s: %v", cPath, err)
		}
		if cPath != GetConversationDir(conf) {
			if f.IsDir() {
				log.Printf("Found conversation %s\n", cPath)

				// create the outbox directory in tmp, then (atomically) move it to outbox
				tmpDir, err := GetUniqueTmpDir(conf)
				if err != nil {
					return err
				}
				defer os.RemoveAll(tmpDir)
				conversationInfo, err := os.Stat(cPath)
				if err != nil {
					// skip this conversation; can't read it
					log.Printf("Error reading permissions on a conversation directory %s", cPath)
					return nil
				}
				var c_perm = conversationInfo.Mode()
				metadataFile := filepath.Join(cPath, MetadataFileName)
				metadataInfo, err := os.Stat(metadataFile)
				if err != nil {
					// skip this conversation; it probably doesn't have a metadata file
					log.Printf("Error reading permissions on metadata file %s", metadataFile)
					return nil
				}
				var m_perm = metadataInfo.Mode()
				oldUmask := syscall.Umask(0000)
				defer syscall.Umask(oldUmask)
				os.Mkdir(filepath.Join(tmpDir, path.Base(cPath)), c_perm)
				err = Copy(metadataFile, filepath.Join(tmpDir, path.Base(cPath), MetadataFileName), m_perm)
				if err != nil {
					log.Printf("Error, can't copy metadata file to temp: %s", metadataFile)
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

func WatchFs(conf config.Config, initFn filepath.WalkFunc, updateFn filepath.WalkFunc) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	// watch initial directory structure
	registerDirectory := getRegisterDirectoryFunction(watcher)
	doInit := func(path string, f os.FileInfo, err error) error {
		err = registerDirectory(path, f, err)
		err = initFn(path, f, err)
		return err
	}

	doUpdate := func(path string, f os.FileInfo, err error) error {
		err = registerDirectory(path, f, err)
		err = updateFn(path, f, err)
		return err
	}

	err = filepath.Walk(conf.RootDir, doInit)

	for {
		select {
		case ev := <-watcher.Event:
			// event in the directory structure; watch any new directories
			log.Println("event:", ev)
			if !(ev.IsDelete() || ev.IsRename()) {
				err = filepath.Walk(ev.Name, doUpdate)
			}
		case err := <-watcher.Error:
			log.Println("error:", err)
		}
	}
}

func getRegisterDirectoryFunction(watcher *fsnotify.Watcher) func(string, os.FileInfo, error) error {
	return func(path string, f os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error in walking over %s: %v", path, err)
		}
		if f.IsDir() {
			log.Printf("Watching %s\n", path)
			err = watcher.Watch(path)
			if err != nil {
				log.Printf("Error watching %s: %v", path, err)
			}
		}
		return nil
	}
}
