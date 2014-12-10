package main

import (
	"github.com/andres-erbsen/chatterbox/client/daemon"
	"log"
)

func main() {
	//TODO read the directory as an argument
	err := daemon.Run("/tmp/foo/bar")
	if err != nil {
		log.Fatal(err)
		return
	}
}
