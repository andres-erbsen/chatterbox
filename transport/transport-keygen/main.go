package main

import (
	"crypto/rand"
	"golang.org/x/crypto/nacl/box"
	"io/ioutil"
	"log"
	"fmt"
	"path"
	"os"
)

const PRIVATE_KEY string = "private_key"
const PUBLIC_KEY string = "public_key"

func main() {
	var dir string
	if len(os.Args) < 2 {
		dir = "."
	} else {
		dir = os.Args[1]
	}

	privfile := path.Join(dir, PRIVATE_KEY)
	pubfile := path.Join(dir, PUBLIC_KEY)

	if _, err := os.Stat(privfile); err == nil {
		fmt.Fprintf(os.Stderr, "%s already exists\n", privfile)
		os.Exit(1)
	}
	if _, err := os.Stat(pubfile); err == nil {
		fmt.Fprintf(os.Stderr, "%s already exists\n", pubfile)
		os.Exit(1)
	}

	pk, sk, err := box.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	if err := ioutil.WriteFile(privfile, sk[:], 0600); err != nil {
		log.Fatal(err)
		os.Exit(3)
	}
	if err := ioutil.WriteFile(pubfile, pk[:], 0644); err != nil {
		log.Fatal(err)
		os.Exit(4)
	}
}
