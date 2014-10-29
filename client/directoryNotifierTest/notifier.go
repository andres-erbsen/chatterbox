// Test file for watching changes to a directory. Watches a root directory and all sub-directories.
// If new directories are created in watched directories those are watched (recursively) as well.
package main

import (
	"code.google.com/p/go.exp/fsnotify"
	"log"
	"os"
	"path/filepath"
)

const rootPath = "/tmp/foo"

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	// watch initial directory structure
	registerDirectory := getRegisterDirectoryFunction(watcher)
	err = filepath.Walk(rootPath, registerDirectory)

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
