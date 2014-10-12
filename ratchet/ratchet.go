// Copyright (c) 2013 Adam Langley. All rights reserved
// Copyright (c) 2014 Andres Erbsen

// Package ratchet implements the axolotl ratchet, by Trevor Perrin. See
// https://github.com/trevp/axolotl/wiki.
//
// This implementation is designed to be used with asynchronous key exchange.
// In particular, it is admitted that two separate axolotl sessions may be
// established between the same two parties, and it is the application's
// responsibility to close one of them.
//
// The key exchange is assumed to be externally authenticated and no identity
// key verification (or exchange) is performed.
package ratchet

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"hash"
	"io"
	"time"

	"code.google.com/p/go.crypto/curve25519"
	"code.google.com/p/go.crypto/nacl/secretbox"
)

type Ratchet struct {
	// rootKey gets updated by the DH ratchet.
	rootKey [32]byte
	// Header keys are used to encrypt message headers.
	sendHeaderKey, recvHeaderKey         [32]byte
	nextSendHeaderKey, nextRecvHeaderKey [32]byte
	// Chain keys are used for forward secrecy updating.
	sendChainKey, recvChainKey [32]byte
	sendCount, recvCount       uint32
	prevSendCount              uint32
	// ratchet is true if we will send a new ratchet value in the next message.
	ratchet                               bool
	ourRatchetPrivate, theirRatchetPublic [32]byte
	// saved is a map from a header key to a map from sequence number to
	// message key.
	saved map[[32]byte]map[uint32]savedKey

	// ourAuthPrivate is updated together with ourRatchetPrivate, but not flushed
	ourAuthPrivate, prevAuthPrivate, theirAuthPublic [32]byte

	rand io.Reader
	now  func() time.Time
}

// savedKey contains a message key and timestamp for a message which has not
// been received. The timestamp comes from the message by which we learn of the
// missing message.
type savedKey struct {
	key       [32]byte
	authPriv  [32]byte
	timestamp time.Time
}

func (r *Ratchet) randBytes(buf []byte) {
	rnd := rand.Reader
	if r.rand != nil {
		rnd = r.rand
	}
	if _, err := io.ReadFull(rnd, buf); err != nil {
		panic(err)
	}
}

// deriveKey takes an HMAC object with key k and a label and calculates out =
// HMAC(k, label). Any prior input to h is ignored and reset.
func deriveKey(out *[32]byte, label []byte, h hash.Hash) {
	h.Reset()
	h.Write(label)
	n := h.Sum(out[:0])
	if &n[0] != &out[0] {
		panic("hash function too large")
	}
}

// These constants are used as the label argument to deriveKey to derive
// independent keys from a master key.
var (
	chainKeyLabel          = []byte("chain key")
	headerKeyLabel         = []byte("header key")
	nextRecvHeaderKeyLabel = []byte("next receive header key")
	rootKeyLabel           = []byte("root key")
	rootKeyUpdateLabel     = []byte("root key update")
	sendHeaderKeyLabel     = []byte("next send header key")
	messageKeyLabel        = []byte("message key")
	chainKeyStepLabel      = []byte("chain key step")
)

const (
	authSize = 16 // in the very beginning of each message
	// handshakePreHeaderSize bytes are added before the header of the first message
	handshakePreHeaderSize = 32 // sender's ephemeral curve25519 pk
	// headerSize is the size, in bytes, of a header's plaintext contents.
	headerSize = 4 /* uint32 message count */ +
		4 /* uint32 previous message count */ +
		32 /* curve25519 ratchet public */ +
		32 /* curve25519 auth public */ +
		24 /* nonce for message */
	// sealedHeader is the size, in bytes, of an encrypted header.
	sealedHeaderSize = 24 /* nonce */ + headerSize + secretbox.Overhead
	// nonceInHeaderOffset is the offset of the message nonce in the
	// header's plaintext.
	nonceInHeaderOffset = 4 + 4 + 32 + 32
	// maxMissingMessages is the maximum number of missing messages that
	// we'll keep track of.
	maxMissingMessages = 8
)

func EncryptFirst(out, msg []byte, theirRatchetPublic *[32]byte, rand io.Reader, now func() time.Time) (*Ratchet, []byte) {
	r := &Ratchet{saved: make(map[[32]byte]map[uint32]savedKey),
		rand:    rand,
		now:     now,
		ratchet: true,
	}
	r.randBytes(r.ourRatchetPrivate[:])
	copy(r.theirRatchetPublic[:], theirRatchetPublic[:])

	var sharedKey [32]byte
	curve25519.ScalarMult(&sharedKey, &r.ourRatchetPrivate, &r.theirRatchetPublic)
	h := hmac.New(sha256.New, sharedKey[:])
	deriveKey(&r.rootKey, rootKeyLabel, h)
	deriveKey(&r.recvHeaderKey, headerKeyLabel, h)
	deriveKey(&r.nextSendHeaderKey, sendHeaderKeyLabel, h)
	deriveKey(&r.nextRecvHeaderKey, nextRecvHeaderKeyLabel, h)
	deriveKey(&r.recvChainKey, chainKeyLabel, h)

	var ourRatchetPublic [32]byte
	curve25519.ScalarBaseMult(&ourRatchetPublic, &r.ourRatchetPrivate)
	tag_idx := len(out)
	out = append(out, make([]byte, authSize)...)
	out = append(out, ourRatchetPublic[:]...)
	out = r.encrypt(out, msg)
	r.fillAuth(out[tag_idx:][:authSize], out[tag_idx+authSize:], theirRatchetPublic)
	return r, out
}

func DecryptFirst(ciphertext []byte, ourRatchetPrivate *[32]byte, rand io.Reader, now func() time.Time) (*Ratchet, []byte, error) {
	if len(ciphertext) < authSize+handshakePreHeaderSize+headerSize {
		return nil, nil, errors.New("first message too short")
	}
	r := &Ratchet{saved: make(map[[32]byte]map[uint32]savedKey),
		rand: rand,
		now:  now,
	}
	copy(r.ourRatchetPrivate[:], ourRatchetPrivate[:])
	copy(r.ourAuthPrivate[:], ourRatchetPrivate[:])

	tag := ciphertext[:authSize]
	var sharedKey [32]byte
	copy(r.theirRatchetPublic[:], ciphertext[authSize:][:handshakePreHeaderSize])
	curve25519.ScalarMult(&sharedKey, &r.ourRatchetPrivate, &r.theirRatchetPublic)
	h := hmac.New(sha256.New, sharedKey[:])
	deriveKey(&r.rootKey, rootKeyLabel, h)
	deriveKey(&r.sendHeaderKey, headerKeyLabel, h)
	deriveKey(&r.nextRecvHeaderKey, sendHeaderKeyLabel, h)
	deriveKey(&r.nextSendHeaderKey, nextRecvHeaderKeyLabel, h)
	deriveKey(&r.sendChainKey, chainKeyLabel, h)

	msg, err := r.decryptAndCheckAuth(tag, ciphertext[authSize:], ciphertext[authSize+handshakePreHeaderSize:])
	return r, msg, err
}

// encrypt acts like append() but appends an encrypted version of msg to out.
func (r *Ratchet) encrypt(out, msg []byte) []byte {
	if r.ratchet {
		r.randBytes(r.ourRatchetPrivate[:])
		copy(r.prevAuthPrivate[:], r.ourAuthPrivate[:])
		r.randBytes(r.ourAuthPrivate[:])
		copy(r.sendHeaderKey[:], r.nextSendHeaderKey[:])

		var sharedKey, keyMaterial [32]byte
		curve25519.ScalarMult(&sharedKey, &r.ourRatchetPrivate, &r.theirRatchetPublic)
		sha := sha256.New()
		sha.Write(rootKeyUpdateLabel)
		sha.Write(r.rootKey[:])
		sha.Write(sharedKey[:])

		sha.Sum(keyMaterial[:0])
		h := hmac.New(sha256.New, keyMaterial[:])
		deriveKey(&r.rootKey, rootKeyLabel, h)
		deriveKey(&r.nextSendHeaderKey, sendHeaderKeyLabel, h)
		deriveKey(&r.sendChainKey, chainKeyLabel, h)

		r.prevSendCount, r.sendCount = r.sendCount, 0
		r.ratchet = false
	}

	var messageKey [32]byte
	h := hmac.New(sha256.New, r.sendChainKey[:])
	deriveKey(&messageKey, messageKeyLabel, h)
	deriveKey(&r.sendChainKey, chainKeyStepLabel, h)

	var ourRatchetPublic, ourAuthPublic [32]byte
	curve25519.ScalarBaseMult(&ourRatchetPublic, &r.ourRatchetPrivate)
	curve25519.ScalarBaseMult(&ourAuthPublic, &r.ourAuthPrivate)
	var header [headerSize]byte
	var headerNonce, messageNonce [24]byte
	r.randBytes(headerNonce[:])
	r.randBytes(messageNonce[:])

	binary.LittleEndian.PutUint32(header[0:4], r.sendCount)
	binary.LittleEndian.PutUint32(header[4:8], r.prevSendCount)
	copy(header[8:], ourRatchetPublic[:])
	copy(header[8+32:], ourAuthPublic[:])
	copy(header[nonceInHeaderOffset:], messageNonce[:])
	out = append(out, headerNonce[:]...)
	out = secretbox.Seal(out, header[:], &headerNonce, &r.sendHeaderKey)
	r.sendCount++
	return secretbox.Seal(out, msg, &messageNonce, &messageKey)
}

// Encrypt acts like append() but appends an encrypted and authenticated version of msg to out.
func (r *Ratchet) Encrypt(out, msg []byte) []byte {
	tag_idx := len(out)
	out = append(out, make([]byte, authSize)...)
	out = r.encrypt(out, msg)
	r.fillAuth(out[tag_idx:][:authSize], out[tag_idx+authSize:], &r.theirAuthPublic)
	return out
}

// trySavedKeys tries to decrypt ciphertext using keys saved for missing messages.
func (r *Ratchet) trySavedKeysAndCheckAuth(authTag, authBody, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < sealedHeaderSize {
		return nil, errors.New("ratchet: header too small to be valid")
	}

	sealedHeader := ciphertext[:sealedHeaderSize]
	var nonce [24]byte
	copy(nonce[:], sealedHeader)
	sealedHeader = sealedHeader[len(nonce):]

	for headerKey, messageKeys := range r.saved {
		header, ok := secretbox.Open(nil, sealedHeader, &nonce, &headerKey)
		if !ok {
			continue
		}
		if len(header) != headerSize {
			continue
		}
		msgNum := binary.LittleEndian.Uint32(header[:4])
		msgKey, ok := messageKeys[msgNum]
		if !ok {
			// This is a fairly common case: the message key might
			// not have been saved because it's the next message
			// key.
			return nil, nil
		}

		sealedMessage := ciphertext[sealedHeaderSize:]
		copy(nonce[:], header[nonceInHeaderOffset:])
		msg, ok := secretbox.Open(nil, sealedMessage, &nonce, &msgKey.key)
		if !ok {
			return nil, errors.New("ratchet: corrupt message")
		}
		if err := r.checkAuth(authTag, authBody, &msgKey.authPriv); err != nil {
			return nil, err
		}
		delete(messageKeys, msgNum)
		if len(messageKeys) == 0 {
			delete(r.saved, headerKey)
		}
		return msg, nil
	}

	return nil, nil
}

// saveKeys takes a header key, the current chain key, a received message
// number and the expected message number and advances the chain key as needed.
// It returns the message key for given given message number and the new chain
// key. If any messages have been skipped over, it also returns savedKeys, a
// map suitable for merging with r.saved, that contains the message keys for
// the missing messages.
func (r *Ratchet) saveKeys(headerKey, recvChainKey *[32]byte, messageNum, receivedCount uint32, authPrivate *[32]byte) (provisionalChainKey, messageKey [32]byte, savedKeys map[[32]byte]map[uint32]savedKey, err error) {
	if messageNum < receivedCount {
		// This is a message from the past, but we didn't have a saved
		// key for it, which means that it's a duplicate message or we
		// expired the save key.
		err = errors.New("ratchet: duplicate message or message delayed longer than tolerance")
		return
	}

	missingMessages := messageNum - receivedCount
	if missingMessages > maxMissingMessages {
		err = errors.New("ratchet: message exceeds reordering limit")
		return
	}

	// messageKeys maps from message number to message key.
	var messageKeys map[uint32]savedKey
	var now time.Time
	if missingMessages > 0 {
		messageKeys = make(map[uint32]savedKey)
		if r.now == nil {
			now = time.Now()
		} else {
			now = r.now()
		}
	}

	copy(provisionalChainKey[:], recvChainKey[:])

	for n := receivedCount; n <= messageNum; n++ {
		h := hmac.New(sha256.New, provisionalChainKey[:])
		deriveKey(&messageKey, messageKeyLabel, h)
		deriveKey(&provisionalChainKey, chainKeyStepLabel, h)
		if n < messageNum {
			messageKeys[n] = savedKey{messageKey, *authPrivate, now}
		}
	}

	if messageKeys != nil {
		savedKeys = make(map[[32]byte]map[uint32]savedKey)
		savedKeys[*headerKey] = messageKeys
	}

	return
}

// mergeSavedKeys takes a map of saved message keys from saveKeys and merges it
// into r.saved.
func (r *Ratchet) mergeSavedKeys(newKeys map[[32]byte]map[uint32]savedKey) {
	for headerKey, newMessageKeys := range newKeys {
		messageKeys, ok := r.saved[headerKey]
		if !ok {
			r.saved[headerKey] = newMessageKeys
			continue
		}

		for n, messageKey := range newMessageKeys {
			messageKeys[n] = messageKey
		}
	}
}

// isZeroKey returns true if key is all zeros.
func isZeroKey(key *[32]byte) bool {
	var x uint8
	for _, v := range key {
		x |= v
	}

	return x == 0
}

func (r *Ratchet) decryptAndCheckAuth(authTag, authBody, ciphertext []byte) ([]byte, error) {
	msg, err := r.trySavedKeysAndCheckAuth(authTag, authBody, ciphertext)
	if err != nil || msg != nil {
		return msg, err
	}

	sealedHeader := ciphertext[:sealedHeaderSize]
	sealedMessage := ciphertext[sealedHeaderSize:]
	var nonce [24]byte
	copy(nonce[:], sealedHeader)
	sealedHeader = sealedHeader[len(nonce):]

	header, ok := secretbox.Open(nil, sealedHeader, &nonce, &r.recvHeaderKey)
	ok = ok && !isZeroKey(&r.recvHeaderKey)
	if ok {
		if len(header) != headerSize {
			return nil, errors.New("ratchet: incorrect header size")
		}
		messageNum := binary.LittleEndian.Uint32(header[:4])
		provisionalChainKey, messageKey, savedKeys, err := r.saveKeys(&r.recvHeaderKey, &r.recvChainKey, messageNum, r.recvCount, &r.prevAuthPrivate)
		if err != nil {
			return nil, err
		}

		copy(nonce[:], header[nonceInHeaderOffset:])
		msg, ok := secretbox.Open(nil, sealedMessage, &nonce, &messageKey)
		if !ok {
			return nil, errors.New("ratchet: corrupt message")
		}
		if err := r.checkAuth(authTag, authBody, &r.prevAuthPrivate); err != nil {
			return nil, err
		}

		copy(r.recvChainKey[:], provisionalChainKey[:])
		r.mergeSavedKeys(savedKeys)
		r.recvCount = messageNum + 1
		return msg, nil
	}

	header, ok = secretbox.Open(nil, sealedHeader, &nonce, &r.nextRecvHeaderKey)
	if !ok {
		return nil, errors.New("ratchet: cannot decrypt")
	}
	if len(header) != headerSize {
		return nil, errors.New("ratchet: incorrect header size")
	}

	if r.ratchet {
		return nil, errors.New("ratchet: received message encrypted to next header key without ratchet flag set")
	}

	messageNum := binary.LittleEndian.Uint32(header[:4])
	prevMessageCount := binary.LittleEndian.Uint32(header[4:8])

	_, _, oldSavedKeys, err := r.saveKeys(&r.recvHeaderKey, &r.recvChainKey, prevMessageCount, r.recvCount, &r.prevAuthPrivate)
	if err != nil {
		return nil, err
	}

	var dhPublic, authPublic, sharedKey, rootKey, chainKey, keyMaterial [32]byte
	copy(dhPublic[:], header[8:])
	copy(authPublic[:], header[8+32:])

	curve25519.ScalarMult(&sharedKey, &r.ourRatchetPrivate, &dhPublic)

	sha := sha256.New()
	sha.Write(rootKeyUpdateLabel)
	sha.Write(r.rootKey[:])
	sha.Write(sharedKey[:])

	var rootKeyHMAC hash.Hash

	sha.Sum(keyMaterial[:0])
	rootKeyHMAC = hmac.New(sha256.New, keyMaterial[:])
	deriveKey(&rootKey, rootKeyLabel, rootKeyHMAC)
	deriveKey(&chainKey, chainKeyLabel, rootKeyHMAC)

	provisionalChainKey, messageKey, savedKeys, err := r.saveKeys(&r.nextRecvHeaderKey, &chainKey, messageNum, 0, &r.ourAuthPrivate)
	if err != nil {
		return nil, err
	}

	copy(nonce[:], header[nonceInHeaderOffset:])
	msg, ok = secretbox.Open(nil, sealedMessage, &nonce, &messageKey)
	if !ok {
		return nil, errors.New("ratchet: corrupt message")
	}
	if err := r.checkAuth(authTag, authBody, &r.ourAuthPrivate); err != nil {
		return nil, err
	}

	copy(r.rootKey[:], rootKey[:])
	copy(r.recvChainKey[:], provisionalChainKey[:])
	copy(r.recvHeaderKey[:], r.nextRecvHeaderKey[:])
	deriveKey(&r.nextRecvHeaderKey, sendHeaderKeyLabel, rootKeyHMAC)
	for i := range r.ourRatchetPrivate {
		r.ourRatchetPrivate[i] = 0
	}
	copy(r.theirRatchetPublic[:], dhPublic[:])
	copy(r.theirAuthPublic[:], authPublic[:])

	r.recvCount = messageNum + 1
	r.mergeSavedKeys(oldSavedKeys)
	r.mergeSavedKeys(savedKeys)
	r.ratchet = true

	return msg, nil
}

func (r *Ratchet) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < authSize+sealedHeaderSize {
		return nil, errors.New("ciphertext too short")
	}
	return r.decryptAndCheckAuth(ciphertext[:authSize], ciphertext[authSize:], ciphertext[authSize:])
}

// mergeSavedKeys takes a map of saved message keys from saveKeys and merges it
// into r.saved.
func (r *Ratchet) FlushSavedKeys(now time.Time, lifetime time.Duration) {
	for headerKey, messageKeys := range r.saved {
		for messageNum, savedKey := range messageKeys {
			if now.Sub(savedKey.timestamp) > lifetime {
				for i := range savedKey.key {
					savedKey.key[i] = 0
				}
				for i := range savedKey.authPriv {
					savedKey.authPriv[i] = 0
				}
				delete(messageKeys, messageNum) // safe: http://golang.org/doc/effective_go.html#for
			}
		}
		if len(messageKeys) == 0 {
			delete(r.saved, headerKey) // safe: http://golang.org/doc/effective_go.html#for
		}
	}
}

func (r *Ratchet) fillAuth(tag, msg []byte, theirAuthPublic *[32]byte) {
}

func (r *Ratchet) checkAuth(tag, msg []byte, ourAuthPrivate *[32]byte) error {
	return nil
}
