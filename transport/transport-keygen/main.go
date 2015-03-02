package main

import (
	"crypto/rand"
	"fmt"
	"golang.org/x/crypto/nacl/box"
	"io/ioutil"
	"log"
	"os"
	"path"
)

const SECRET_KEY string = "transport_secret_key"
const PUBLIC_KEY string = "transport_public_key"

func main() {
	var dir string
	if len(os.Args) < 2 {
		dir = "."
	} else {
		dir = os.Args[1]
	}

	skfile := path.Join(dir, SECRET_KEY)
	pkfile := path.Join(dir, PUBLIC_KEY)

	if _, err := os.Stat(skfile); err == nil {
		fmt.Fprintf(os.Stderr, "%s already exists\n", skfile)
		os.Exit(1)
	}
	if _, err := os.Stat(pkfile); err == nil {
		fmt.Fprintf(os.Stderr, "%s already exists\n", pkfile)
		os.Exit(1)
	}

	pk, sk, err := box.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	if err := ioutil.WriteFile(skfile, sk[:], 0600); err != nil {
		log.Fatal(err)
		os.Exit(3)
	}
	if err := ioutil.WriteFile(pkfile, pk[:], 0644); err != nil {
		log.Fatal(err)
		os.Exit(4)
	}
}
