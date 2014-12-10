package main

import (
	"fmt"
	"github.com/andres-erbsen/chatterbox/server"
	"github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 5 {
		fmt.Fprintf(os.Stderr, "USAGE: %s <sk> <pk> <dbdir> <host:port>", os.Args[0])
		os.Exit(2)
	}
	db, err := leveldb.OpenFile(os.Args[3], nil)
	if err != nil {
		log.Fatal(err)
	}
	skBytes, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	pkBytes, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	var sk, pk [32]byte
	copy(sk[:], skBytes)
	copy(pk[:], pkBytes)
	server.StartServer(db, nil, &pk, &sk, os.Args[4])
	select {}
}
