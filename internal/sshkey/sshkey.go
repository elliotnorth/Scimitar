// Package sshkey manages the one SSH key Scimitar uses to log in to network devices.
package sshkey

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

const (
	comment = "scimitar"
	// RSA because every Junos release accepts it; older ones have no ed25519 or ECDSA.
	rsaBits = 3072
)

// LoadOrCreate reads the private key at path, creating a new RSA key there if none exists.
func LoadOrCreate(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		if err := create(path); err != nil {
			return nil, fmt.Errorf("create ssh key: %w", err)
		}
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("read ssh key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("parse ssh key %s: %w", path, err)
	}
	return signer, nil
}

// Load reads an existing private key without creating one.
func Load(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ssh key: %w", err)
	}
	return ssh.ParsePrivateKey(data)
}

func create(path string) error {
	key, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return err
	}
	block, err := ssh.MarshalPrivateKey(key, comment)
	if err != nil {
		return err
	}
	// O_EXCL keeps two processes from racing to write different keys.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if err := pem.Encode(f, block); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	return f.Close()
}

// Replace removes the key at path and creates a new one.
func Replace(path string) (ssh.Signer, error) {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return LoadOrCreate(path)
}

// AuthorizedKey returns the public key as one authorized_keys line, for example "ssh-rsa AAAA... scimitar".
func AuthorizedKey(signer ssh.Signer) string {
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey()))) + " " + comment
}
