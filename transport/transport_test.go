package transport

import (
	"bytes"
	"crypto/rand"
	"net"
	"testing"

	"code.google.com/p/go.crypto/nacl/box"
)

func runHandshake(t *testing.T, haveClient bool) (c1 *Conn, c2 *Conn) {
	ch1, ch2 := make(chan struct{}), make(chan struct{})
	p1, p2 := net.Pipe()
	pk1, sk1, err1 := box.GenerateKey(rand.Reader)
	pk2, sk2, err2 := box.GenerateKey(rand.Reader)
	if err1 != nil || err2 != nil {
		t.Fatal("key generation failed")
	}
	var pk2_1, pk1_2 *[32]byte
	go func() { c1, pk2_1, err1 = Handshake(p1, pk1, sk1, haveClient, 1<<12); close(ch1) }()
	go func() { c2, pk1_2, err2 = Handshake(p2, pk2, sk2, false, 1<<12); defer close(ch2) }()
	<-ch1
	<-ch2
	if err1 != nil {
		t.Fatal(err1)
	}
	if !bytes.Equal(pk2_1[:], pk2[:]) {
		t.Error("1 observed wrong pk")
	}
	if err2 != nil {
		t.Fatal(err2)
	}
	if !bytes.Equal(pk1_2[:], pk1[:]) {
		t.Error("2 observed wrong pk")
	}
	return c1, c2
}

func TestSymmetricHandshake(t *testing.T) {
	c1, c2 := runHandshake(t, false)
	c1.Close()
	c2.Close()
}

func TestClientHandshake(t *testing.T) {
	c1, c2 := runHandshake(t, true)
	c1.Close()
	c2.Close()
}

func TestReadWriteFrame(t *testing.T) {
	c1, c2 := runHandshake(t, false)
	defer c1.Close()
	defer c2.Close()
	for i := 0; i < 10; i++ {
		ch := make(chan struct{})
		go func() {
			defer close(ch)
			n, err := c1.WriteFrame([]byte("fish"))
			if n != 4 {
				t.Error("write: n != 4")
			}
			if err != nil {
				t.Error(err)
			}
		}()
		var buf [8]byte
		n, err := c2.ReadFrame(buf[:])
		if n != 4 {
			t.Error("read: n != 4")
		}
		if err != nil {
			t.Error(err)
		}
		c1, c2 = c2, c1
	}
}
