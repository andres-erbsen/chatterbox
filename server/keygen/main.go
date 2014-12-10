package main

import (
	"crypto/rand"
	"golang.org/x/crypto/nacl/box"
	"os"
)

func main() {
	pk, sk, _ := box.GenerateKey(rand.Reader)
	os.Stdout.Write(pk[:])
	os.Stderr.Write(sk[:])
}
