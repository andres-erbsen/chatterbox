package main

import (
	"github.com/andres-erbsen/chatterbox/client/daemon"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	daemon, err := daemon.New(os.Args[1])
	if err != nil {
		log.Fatal(err)
		return
	}

	daemon.Start()

	s := make(chan os.Signal)
	signal.Notify(s, os.Kill, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)
	<-s
	daemon.Stop()
}
