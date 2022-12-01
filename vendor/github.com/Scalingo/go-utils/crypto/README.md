# Crypto Tooling

This package `github.com/Scalingo/go-utils/crypto` aims at providing common crypto primitive helpers.

## Secret Generator

```go
// Generate keys with different formats
crypto.CreateKey(size int) ([]byte, error)
crypto.CreateKeyString(size int) (string, error)
crypto.CreateKeyBase64String(size int) (string, error)

// Parse hex-string key back to binary
crypto.ParseKey(key string) ([]byte, error)
```

## Symmetric Block Encryption (AES-CFB)

```go
crypto.Encrypt(key, plaintext []byte) ([]byte, error)
crypto.Decrypt(key, ciphertext []byte) ([]byte, error)
```

## HMAC-SHA Signature

```go
crypto.HMAC256(key, payload []byte) ([]byte, error)
crypto.HMAC512(key, payload []byte) ([]byte, error)
```

## Data Stream Encryption (AES-256-CTR)

```go
crypto.NewStreamEncrypter(encryptionKey, hmacKey []byte, plaintext io.Reader) (*StreamEncrypter, error)
crypto.NewStreamDecrypter(encryptionKey, hmacKey []byte, ciphertext io.Reader) (*StreamDecrypter, error)
```

* Both `StreamEncrypter` and `StreamDecrypter` are `io.Reader`
* Calling `Read` on them will be blocking if no input is provided
* They'll return `io.EOF` once the input returns `io.EOF`.

