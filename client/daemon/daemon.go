// client daemon
//   watches the file system for new messages --> sends them
//   communicates with the server --> receive new messages
package main

import (
	"github.com/andres-erbsen/chatterbox/client/util/config"
	"github.com/andres-erbsen/chatterbox/client/util/filesystem"
	"log"
	"os"
	"time"
)

func GetRootDir() string {
	const temporaryConstantRootDirectory = "/tmp/foo/bar"
	return temporaryConstantRootDirectory
}

func main() {
	conf := config.Config{
		RootDir:    GetRootDir(),
		Time:       time.Now,
		TempPrefix: "daemon",
	}
	filesystem.InitFs(conf)

	initFn := func(path string, f os.FileInfo, err error) error {
		log.Printf("init path: %s\n", path)
		return err
	}

	updateFn := func(path string, f os.FileInfo, err error) error {
		log.Printf("update path: %s\n", path)
		return err
	}

	filesystem.WatchFs(conf, initFn, updateFn)
}
