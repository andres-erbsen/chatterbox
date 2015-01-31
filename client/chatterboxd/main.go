package main

import (
	"github.com/andres-erbsen/chatterbox/client/daemon"
	"log"
	"os"
	"os/signal"
)

func main() {
	daemon, err := daemon.New(os.Args[1])
	if err != nil {
		log.Fatal(err)
		return
	}

	shutdown := make(chan struct{})
	go func() {
		ch := make(chan os.Signal)
		signal.Notify(ch, os.Kill, os.Interrupt)
		<-ch
		close(shutdown)
	}()
	//TODO read the directory as an argument
	err = daemon.Run(shutdown)
	if err != nil {
		log.Fatal(err)
		return
	}
}
