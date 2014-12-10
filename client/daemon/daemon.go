// client daemon
//   watches the file system for new messages --> sends them
//   communicates with the server --> receive new messages
package daemon

import (
	"code.google.com/p/go.exp/fsnotify"
	"log"
	"os"
	"time"
)

func Run(rootDir string, shutdown <-chan struct{}) error {
	conf := Config{
		RootDir:    rootDir,
		Time:       time.Now,
		TempPrefix: "daemon",
	}

	err := InitFs(conf)
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
	defer watcher.Close()

	err = WatchDir(watcher, conf.OutboxDir(), initFn)
	if err != nil {
		return err
	}

	for {
		select {
		case <-shutdown:
			return nil
		case ev := <-watcher.Event:
			// event in the directory structure; watch any new directories
			if !(ev.IsDelete() || ev.IsRename()) {
				err = WatchDir(watcher, ev.Name, initFn)
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
