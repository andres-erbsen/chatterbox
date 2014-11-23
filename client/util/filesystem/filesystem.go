// file system utility directory

package filesystem

import (
	"code.google.com/p/go.exp/fsnotify"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"syscall"
)

func GetConversationDir(rootDir string) string {
	return rootDir + "/conversations"
}

func GetOutboxDir(rootDir string) string {
	return rootDir + "/outbox"
}

func getTmpDir(rootDir string) string {
	return rootDir + "/tmp"
}

func getJournalDir(rootDir string) string {
	return rootDir + "/journal"
}

func GetKeysDir(rootDir string) string {
	return rootDir + "/keys"
}

func GetUiInfoDir(rootDir string) string {
	return rootDir + "/ui_info"
}

func GetUniqueTmpDir(rootDir string) (string, error) {
	return ioutil.TempDir(getTmpDir(rootDir), "")
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
func InitFs(rootDir string) error {
	// create root directory and immediate sub directories
	os.MkdirAll(rootDir, 0700)
	subdirs := []string{
		GetConversationDir(rootDir),
		GetOutboxDir(rootDir),
		getTmpDir(rootDir),
		getJournalDir(rootDir),
		GetKeysDir(rootDir),
		GetUiInfoDir(rootDir),
	}
	for _, dir := range subdirs {
		os.Mkdir(dir, 0700)
	}

	// for each existing conversation, create a folder in the outbox
	copyToOutbox := func(cPath string, f os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error reading conversation directories %s: %v", cPath, err)
		}
		if cPath != GetConversationDir(rootDir) {
			if f.IsDir() {
				log.Printf("Found conversation %s\n", cPath)

				// create the outbox directory in tmp, then (atomically) move it to outbox
				tmpDir, err := GetUniqueTmpDir(rootDir)
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
				metadataFile := cPath + "/" + MetadataFileName
				metadataInfo, err := os.Stat(metadataFile)
				if err != nil {
					// skip this conversation; it probably doesn't have a metadata file
					log.Printf("Error reading permissions on metadata file %s", metadataFile)
					return nil
				}
				var m_perm = metadataInfo.Mode()
				oldUmask := syscall.Umask(0000)
				defer syscall.Umask(oldUmask)
				os.Mkdir(tmpDir+"/"+path.Base(cPath), c_perm)
				err = Copy(metadataFile, tmpDir+"/"+path.Base(cPath)+"/"+MetadataFileName, m_perm)
				if err != nil {
					log.Printf("Error, can't copy metadata file to temp: %s", metadataFile)
					return err
				}
				err = os.Rename(tmpDir+"/"+path.Base(cPath), GetOutboxDir(rootDir)+"/"+path.Base(cPath))
				if err != nil {
					// skip this conversation; this probably means it already exists in the outbox
					return nil
				}
			}
		}
		return nil
	}
	err := filepath.Walk(GetConversationDir(rootDir), copyToOutbox)
	if err != nil {
		return err
	}
	return nil
}

func WatchFs(rootDir string, initFn filepath.WalkFunc, updateFn filepath.WalkFunc) {
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

	err = filepath.Walk(rootDir, doInit)

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
