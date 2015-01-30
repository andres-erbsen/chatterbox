package main

import (
	"crypto/rand"
	"golang.org/x/crypto/nacl/box"
	"os"
)

func main() {
	pk, sk, err := box.GenerateKey(rand.Reader)
	if err != nil {
		os.Exit(2)
	}
	if _, err := os.Stdout.Write(pk[:]); err != nil {
		os.Exit(3)
	}
	if _, err := os.Stderr.Write(sk[:]); err != nil {
		os.Exit(4)
	}
}
