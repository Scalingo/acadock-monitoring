package crypto

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/hex"

	"github.com/pkg/errors"
)

// CreateKey creates a key of a given size by reading that much data off the crypto/rand reader.
func CreateKey(keySize int) ([]byte, error) {
	key := make([]byte, keySize)
	_, err := cryptorand.Read(key)
	if err != nil {
		return nil, errors.Wrap(err, "fail to generate random bytes")
	}
	return key, nil
}

// CreateKeyString generates a new key and returns it as a hex string.
func CreateKeyString(keySize int) (string, error) {
	key, err := CreateKey(keySize)
	if err != nil {
		return "", errors.Wrap(err, "fail to create key")
	}
	return hex.EncodeToString(key), nil
}

// CreateKeyBase64String generates a new key and returns it as a base64 std encoding string.
func CreateKeyBase64String(keySize int) (string, error) {
	key, err := CreateKey(keySize)
	if err != nil {
		return "", errors.Wrap(err, "fail to create key")
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// ParseKey parses a key from an hexadecimal representation.
func ParseKey(key string) ([]byte, error) {
	decoded, err := hex.DecodeString(key)
	if err != nil {
		return nil, errors.Wrap(err, "fail to decode hexa string")
	}
	if len(decoded) != DefaultKeySize {
		return nil, errors.New("parse key; invalid key length")
	}
	return decoded, nil
}
