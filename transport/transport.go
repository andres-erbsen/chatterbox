// Package transport implements an encrypted and authenticated bytestream
// connection. The key-exchange is like SIGMA-I, with the exception that
// instead of unvirersally verifiable signatures, nacl/box between one party's
// ephemeral key and the other's long-term key is used for authentication (this
// provides deniability). Subsequent messages are encrypted using nacl/box with
// the message counter as an implicit nonce.
package transport

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"net"

	"code.google.com/p/go.crypto/nacl/box"
)

// Conn is an ancrypted and authenticated connection that is NOT concurrency-safe
type Conn struct {
	unencrypted           net.Conn
	readNonce, writeNonce uint64
	key                   [32]byte
	readBuf, writeBuf     []byte
	maxFrameSize          int
}

var nullNonce = [24]byte{}

func Handshake(unencrypted net.Conn, pk, sk *[32]byte, amClient bool, maxFrameSize int) (*Conn, *[32]byte, error) {
	ourEphemeralPublic, ourEphemeralSecret, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	var theirEphemeralPublic, theirPK [32]byte
	var readErr, writeErr error
	writeDone, readDone := make(chan struct{}), make(chan chan struct{})
	go func() { _, writeErr = unencrypted.Write(ourEphemeralPublic[:]); close(writeDone) }()
	go func() { _, readErr = io.ReadFull(unencrypted, theirEphemeralPublic[:]); close(readDone) }()
	if <-writeDone; writeErr != nil {
		return nil, nil, writeErr
	}
	if <-readDone; readErr != nil {
		return nil, nil, readErr
	}

	ret := &Conn{unencrypted: unencrypted, maxFrameSize: maxFrameSize,
		readBuf:  make([]byte, binary.MaxVarintLen64+box.Overhead+maxFrameSize),
		writeBuf: make([]byte, binary.MaxVarintLen64+box.Overhead+maxFrameSize)}
	if bytes.Compare(ourEphemeralPublic[:], theirEphemeralPublic[:]) < 0 {
		ret.writeNonce = 1
	} else {
		ret.readNonce = 1
	}
	box.Precompute(&ret.key, &theirEphemeralPublic, ourEphemeralSecret)

	writeDone, readDone = make(chan struct{}), make(chan chan struct{})
	go func() {
		defer close(readDone)
		var theirHandshake [32 + (box.Overhead + 32 + 32)]byte // theirPK, box(theirEphPK, ourEphPK)
		if _, readErr = ret.ReadFrame(theirHandshake[:]); readErr != nil {
			return
		}
		copy(theirPK[:], theirHandshake[:32])
		hs, ok := box.Open(nil, theirHandshake[32:], &nullNonce, &theirPK, ourEphemeralSecret)
		if !ok || !bytes.Equal(hs, append(theirEphemeralPublic[:], ourEphemeralPublic[:]...)) {
			readErr = errors.New("authentication failed")
		}
	}()
	go func() {
		defer close(writeDone)
		ourHandshake := box.Seal(pk[:], append(ourEphemeralPublic[:], theirEphemeralPublic[:]...),
			&nullNonce, &theirEphemeralPublic, sk)
		if amClient {
			if <-readDone; readErr != nil { // only talk to the right server
				return
			}
		}
		_, writeErr = ret.WriteFrame(ourHandshake)
	}()
	if <-readDone; readErr != nil {
		return nil, nil, readErr
	}
	if <-writeDone; writeErr != nil {
		return nil, nil, writeErr
	}
	return ret, &theirPK, nil
}

func (c *Conn) WriteFrame(b []byte) (int, error) {
	if len(b) > c.maxFrameSize {
		return 0, errors.New("write frame too large")
	}
	var nonce [24]byte
	binary.LittleEndian.PutUint64(nonce[:], c.writeNonce)
	c.writeNonce += 2
	i := binary.PutUvarint(c.writeBuf, uint64(box.Overhead+len(b)))
	buf := box.SealAfterPrecomputation(c.writeBuf[:i], b, &nonce, &c.key)
	if _, err := c.unencrypted.Write(buf); err != nil {
		return 0, err
	}
	return len(b), nil
}

type byteReader struct{ io.Reader }

func (r byteReader) ReadByte() (byte, error) {
	var ret [1]byte
	_, err := io.ReadFull(r, ret[:])
	return ret[0], err
}

func (c *Conn) ReadFrame(b []byte) (int, error) {
	var nonce [24]byte
	binary.LittleEndian.PutUint64(nonce[:], c.readNonce)
	c.readNonce += 2
	size, err := binary.ReadUvarint(byteReader{c.unencrypted})
	if err != nil {
		return 0, err
	}
	if _, err := io.ReadFull(c.unencrypted, c.readBuf[:size]); err != nil {
		return 0, err
	}
	b2, ok := box.OpenAfterPrecomputation(b[:0], c.readBuf[:size], &nonce, &c.key)
	if !ok {
		return 0, errors.New("authentication failed")
	}
	if &b[0] != &b2[0] {
		panic("ReadFrame buffer space accounting failed")
	}
	return int(size - box.Overhead), nil
}

func (c *Conn) Close() error {
	for i := 0; i < 32; i++ {
		c.key[i] = 0
	}
	return c.unencrypted.Close()
}
