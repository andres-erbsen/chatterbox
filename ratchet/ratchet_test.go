// Copyright (c) 2013 Adam Langley. All rights reserved
// Copyright (c) 2014 Andres Erbsen

package ratchet

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"

	"golang.org/x/crypto/curve25519"
)

func nowFunc() time.Time {
	var t time.Time
	return t
}

// TODO: test that messages with bad auth are ignored
var dontFillAuth = func([]byte, []byte, *[32]byte) {}
var dontCheckAuth = func([]byte, []byte, []byte, *[32]byte) error { return nil }

func pairedRatchet() (a, b *Ratchet) {
	var preKeyA, preKeyAPrivate [32]byte
	rand.Read(preKeyAPrivate[:])
	curve25519.ScalarBaseMult(&preKeyA, &preKeyAPrivate)

	a = &Ratchet{Now: nowFunc, FillAuth: dontFillAuth, CheckAuth: dontCheckAuth}
	b = &Ratchet{Now: nowFunc, FillAuth: dontFillAuth, CheckAuth: dontCheckAuth}
	msgBtoA := b.EncryptFirst(nil, nil, &preKeyA)
	_, err := a.DecryptFirst(msgBtoA, &preKeyAPrivate)
	if err != nil {
		panic(err)
	}
	return
}

func TestExchange(t *testing.T) {
	a, b := pairedRatchet()

	msg := []byte("test message")
	encrypted := a.Encrypt(nil, msg)
	result, err := b.Decrypt(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(msg, result) {
		t.Fatalf("result doesn't match: %x vs %x", msg, result)
	}
}

type scriptAction struct {
	// object is one of sendA, sendB or sendDelayed. The first two options
	// cause a message to be sent from one party to the other. The latter
	// causes a previously delayed message, identified by id, to be
	// delivered.
	object int
	// result is one of deliver, drop or delay. If delay, then the message
	// is stored using the value in id. This value can be repeated later
	// with a sendDelayed.
	result int
	id     int
}

const (
	sendA = iota
	sendB
	sendDelayed
	deliver
	drop
	delay
)

func reinitRatchet(t *testing.T, r *Ratchet) *Ratchet {
	r.FlushSavedKeys(nowFunc(), 1*time.Hour)
	data, err := r.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal: %s", err)
	}
	newR := &Ratchet{Now: r.Now, Rand: r.Rand, FillAuth: r.FillAuth, CheckAuth: r.CheckAuth}
	if err := newR.Unmarshal(data); err != nil {
		t.Fatalf("Failed to unmarshal: %s", err)
	}
	return newR

}

func testScript(t *testing.T, script []scriptAction) {
	type delayedMessage struct {
		msg       []byte
		encrypted []byte
		fromA     bool
	}
	delayedMessages := make(map[int]delayedMessage)
	a, b := pairedRatchet()

	for i, action := range script {
		switch action.object {
		case sendA, sendB:
			sender, receiver := a, b
			if action.object == sendB {
				sender, receiver = receiver, sender
			}

			var msg [20]byte
			rand.Reader.Read(msg[:])
			encrypted := sender.Encrypt(nil, msg[:])

			switch action.result {
			case deliver:
				result, err := receiver.Decrypt(encrypted)
				if err != nil {
					t.Fatalf("#%d: receiver returned error: %s", i, err)
				}
				if !bytes.Equal(result, msg[:]) {
					t.Fatalf("#%d: bad message: got %x, not %x", i, result, msg[:])
				}
			case delay:
				if _, ok := delayedMessages[action.id]; ok {
					t.Fatalf("#%d: already have delayed message with id %d", i, action.id)
				}
				delayedMessages[action.id] = delayedMessage{msg[:], encrypted, sender == a}
			case drop:
			}
		case sendDelayed:
			delayed, ok := delayedMessages[action.id]
			if !ok {
				t.Fatalf("#%d: no such delayed message id: %d", i, action.id)
			}

			receiver := a
			if delayed.fromA {
				receiver = b
			}

			result, err := receiver.Decrypt(delayed.encrypted)
			if err != nil {
				t.Fatalf("#%d: receiver returned error: %s", i, err)
			}
			if !bytes.Equal(result, delayed.msg) {
				t.Fatalf("#%d: bad message: got %x, not %x", i, result, delayed.msg)
			}
		}

		a = reinitRatchet(t, a)
		b = reinitRatchet(t, b)
	}
}

func TestBackAndForth(t *testing.T) {
	testScript(t, []scriptAction{
		{sendA, deliver, -1},
		{sendB, deliver, -1},
		{sendA, deliver, -1},
		{sendB, deliver, -1},
		{sendA, deliver, -1},
		{sendB, deliver, -1},
	})
}

func TestReorder(t *testing.T) {
	testScript(t, []scriptAction{
		{sendA, deliver, -1},
		{sendA, delay, 0},
		{sendA, deliver, -1},
		{sendDelayed, deliver, 0},
	})
}

func TestReorderAfterRatchet(t *testing.T) {
	testScript(t, []scriptAction{
		{sendA, deliver, -1},
		{sendA, delay, 0},
		{sendB, deliver, -1},
		{sendA, deliver, -1},
		{sendB, deliver, -1},
		{sendDelayed, deliver, 0},
	})
}

func TestReorderMore(t *testing.T) {
	testScript(t, []scriptAction{
		{sendA, deliver, -1},
		{sendA, delay, 0},
		{sendB, delay, 1},
		{sendA, delay, 2},
		{sendB, delay, 3},
		{sendA, deliver, -1},
		{sendB, deliver, -1},
		{sendDelayed, deliver, 2},
		{sendDelayed, deliver, 1},
		{sendDelayed, deliver, 3},
		{sendDelayed, deliver, 0},
	})
}

func TestDrop(t *testing.T) {
	testScript(t, []scriptAction{
		{sendA, drop, -1},
		{sendA, drop, -1},
		{sendA, drop, -1},
		{sendA, drop, -1},
		{sendA, deliver, -1},
		{sendB, deliver, -1},
	})
}

func TestOverhead(t *testing.T) {
	var preKeyA, preKeyAPrivate [32]byte
	rand.Read(preKeyAPrivate[:])
	curve25519.ScalarBaseMult(&preKeyA, &preKeyAPrivate)

	a := &Ratchet{Now: nowFunc, FillAuth: dontFillAuth, CheckAuth: dontCheckAuth}
	b := &Ratchet{Now: nowFunc, FillAuth: dontFillAuth, CheckAuth: dontCheckAuth}
	encryptedFirst := b.EncryptFirst(nil, nil, &preKeyA)
	if len(encryptedFirst) != OverheadFirst {
		t.Errorf("expected first message overhead overhead %d, got %d", OverheadFirst, len(encryptedFirst))
	}
	if _, err := a.DecryptFirst(encryptedFirst, &preKeyAPrivate); err != nil {
		t.Fatal(err)
	}

	r, _ := pairedRatchet()
	encrypted := r.Encrypt(nil, nil)
	if len(encrypted) != Overhead {
		t.Errorf("expected subsequent message overhead %d, got %d", Overhead, len(encrypted))
	}
}
