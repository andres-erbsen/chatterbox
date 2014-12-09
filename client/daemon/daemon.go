// client daemon
//   watches the file system for new messages --> sends them
//   communicates with the server --> receive new messages
package main

import (
	"code.google.com/p/go.exp/fsnotify"
	"github.com/andres-erbsen/chatterbox/client/daemon/config"
	"github.com/andres-erbsen/chatterbox/client/daemon/filesystem"
	"log"
	"os"
	"time"
)

func GetRootDir() string {
	const temporaryConstantRootDirectory = "/tmp/foo/bar"
	return temporaryConstantRootDirectory
}

func main() {
	err := mainReturnErr()
	if err != nil {
		log.Fatal(err)
		return
	}
}

func mainReturnErr() error {
	conf := config.Config{
		RootDir:    GetRootDir(),
		Time:       time.Now,
		TempPrefix: "daemon",
	}

	err := filesystem.InitFs(conf)
	if err != nil {
		return err
	}

	initFn := func(path string, f os.FileInfo, err error) error {
		log.Printf("init path: %s\n", path)
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	err = filesystem.WatchDir(watcher, filesystem.GetOutboxDir(conf), initFn)
	if err != nil {
		return err
	}

	for {
		select {
		case ev := <-watcher.Event:
			// event in the directory structure; watch any new directories
			if !(ev.IsDelete() || ev.IsRename()) {
				err = filesystem.WatchDir(watcher, ev.Name, initFn)
				if err != nil {
					return err
				}
			}
		case err := <-watcher.Error:
			if err != nil {
				return err
			}
		}
	}

}
