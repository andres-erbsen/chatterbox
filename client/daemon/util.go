package daemon

import (
	"golang.org/x/crypto/nacl/box"
	"crypto/rand"
)

// returns public, secret, error
func GeneratePrekeys(numKeys int) ([]*[32]byte, []*[32]byte, error) {
	secret := make([]*[32]byte, numKeys)
	public := make([]*[32]byte, numKeys)

	for i := 0; i < numKeys; i++ {
		pkAuth, skAuth, err := box.GenerateKey(rand.Reader)
		if err != nil {
			return nil, nil, err
		}

		public[i] = pkAuth
		secret[i] = skAuth
	}

	return public, secret, nil
}
