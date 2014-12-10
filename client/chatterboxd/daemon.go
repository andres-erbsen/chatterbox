package main

import (
	"github.com/andres-erbsen/chatterbox/client/daemon"
	"log"
	"os"
	"os/signal"
)

func main() {
	shutdown := make(chan struct{})
	go func() {
		ch := make(chan os.Signal)
		signal.Notify(ch, os.Kill, os.Interrupt)
		<-ch
		close(shutdown)
	}()
	//TODO read the directory as an argument
	err := daemon.Run("/tmp/foo/bar", shutdown)
	if err != nil {
		log.Fatal(err)
		return
	}
}
