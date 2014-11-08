// file system utility directory

package filesystem

import (
	"code.google.com/p/go.exp/fsnotify"
	"log"
	"os"
	"path"
	"path/filepath"
	"syscall"
)

func GetRootDir() string {
	const temporaryConstantRootDirectory = "/tmp/foo/bar"
	return temporaryConstantRootDirectory
}

func GetConversationDir(rootDir string) string {
	return rootDir + "/conversations"
}

func GetOutboxDir(rootDir string) string {
	return rootDir + "/outbox"
}

func GetTmpDir(rootDir string) string {
	return rootDir + "/tmp"
}

func GetKeysDir(rootDir string) string {
	return rootDir + "/keys"
}

func GetUiInfoDir(rootDir string) string {
	return rootDir + "/ui_info"
}

func InitFs(rootDir string) error {
	// create root directory and immediate sub directories
	os.MkdirAll(rootDir, 0700)
	subdirs := []string{
		GetConversationDir(rootDir),
		GetOutboxDir(rootDir),
		GetTmpDir(rootDir),
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

				fileInfo, err := os.Stat(cPath)
				if err != nil {
					log.Printf("Error reading permissions on a conversation directory %s", cPath)
				}
				var perm = fileInfo.Mode()
				oldUmask := syscall.Umask(0000)
				os.Mkdir(GetOutboxDir(rootDir)+"/"+path.Base(cPath), perm)
				syscall.Umask(oldUmask)
				// TODO figure out how metadata works and if that needs to be copied too
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

func WatchFs(rootDir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	// watch initial directory structure
	registerDirectory := getRegisterDirectoryFunction(watcher)
	err = filepath.Walk(rootDir, registerDirectory)

	for {
		select {
		case ev := <-watcher.Event:
			// event in the directory structure; watch any new directories
			log.Println("event:", ev)
			if !ev.IsDelete() {
				err = filepath.Walk(ev.Name, registerDirectory)
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
