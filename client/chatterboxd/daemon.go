package main

import (
	"github.com/andres-erbsen/chatterbox/client/daemon"
	"log"
)

func main() {
	err := daemon.Run()
	if err != nil {
		log.Fatal(err)
		return
	}
}
