package main

import (
	"code.google.com/p/gogoprotobuf/io"
	"testing"
)

func TestAccountCreation(t *testing.T) {
	err = RunServer()
	if err != nil {
		t.Error(err)
	}
	conn, err := net.Dial("tcp", "localhost:8888")
	if err != nil {
		t.Error(err)
	}
	writer := io.
		t.Error("this test sucks")
}
