package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
)

const (
	iterations = 600_000
	keyLength  = 32
	saltLength = 16
)

type PasswordHash struct {
	Salt       []byte `json:"salt"`
	Iterations int    `json:"iterations"`
	Hash       []byte `json:"hash"`
}

func HashPassword(password string) (PasswordHash, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return PasswordHash{}, err
	}
	hash, err := pbkdf2.Key(sha256.New, password, salt, iterations, keyLength)
	if err != nil {
		return PasswordHash{}, err
	}
	return PasswordHash{Salt: salt, Iterations: iterations, Hash: hash}, nil
}

func (h PasswordHash) Matches(password string) bool {
	got, err := pbkdf2.Key(sha256.New, password, h.Salt, h.Iterations, len(h.Hash))
	return err == nil && subtle.ConstantTimeCompare(got, h.Hash) == 1
}

func randomToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
