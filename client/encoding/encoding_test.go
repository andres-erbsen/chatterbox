package encoding

import (
	"crypto/rand"
	"testing"
	"unicode/utf8"
)

func TestEncodeKnown(t *testing.T) {
	r := EscapeFilename("$2 for 10% / $3 for 20%\x01")
	if r != "$2 for 10%percent %slash $3 for 20%percent%01" {
		t.Fatal(r)
	}
}

func TestEncodeKnownHex(t *testing.T) {
	r := EscapeFilename("\x1b")
	if r != "%1b" {
		t.Fatalf("encoded as: %x", r)
	}
	d, err := UnescapeFilename("%1b")
	if err != nil {
		t.Error(err)
	}
	if d != "\x1b" {
		t.Errorf("Decoded to: %x", d)
	}
}

func TestEncodeDecodeRandom(t *testing.T) {
	for i := 0; i < 100; {
		var b [1]byte
		rand.Read(b[:])
		bs := make([]byte, int(b[0]))
		rand.Read(bs[:])
		s := string(bs)
		if !utf8.ValidString(s) {
			continue
		}
		i++
		es := EscapeFilename(s)
		ss, err := UnescapeFilename(es)
		if err != nil {
			t.Error(err)
		}
		if s != ss {
			t.Errorf("not invertible:\n%x : %s\n%x : %s\n%x : %s", s, s, es, es, ss, ss)
		}
	}
}
