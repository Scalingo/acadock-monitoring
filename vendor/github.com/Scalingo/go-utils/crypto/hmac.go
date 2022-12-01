package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
)

// HMAC512 sha512 hashes data with the given key.
func HMAC512(key, plainText []byte) []byte {
	mac := hmac.New(sha512.New, key)
	// Writing in a Hash cannot return an error, so ignoring it.
	// https://cs.opensource.google/go/go/+/refs/tags/go1.19.1:src/crypto/sha512/sha512.go;l=262
	_, _ = mac.Write(plainText)
	return mac.Sum(nil)
}

// HMAC256 sha256 hashes data with the given key.
func HMAC256(key, plainText []byte) []byte {
	mac := hmac.New(sha256.New, key)
	// Writing in a Hash cannot return an error, so ignoring it.
	// https://cs.opensource.google/go/go/+/refs/tags/go1.19.1:src/crypto/sha256/sha256.go;l=191
	_, _ = mac.Write(plainText)
	return mac.Sum(nil)
}
