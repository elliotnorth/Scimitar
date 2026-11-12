// Package sshkey manages the one SSH key Scimitar uses to log in to network devices.
package sshkey

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

const comment = "scimitar"

// LoadOrCreate reads the private key at path, creating a new ed25519 key there if none exists.
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
	_, key, err := ed25519.GenerateKey(rand.Reader)
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

// AuthorizedKey returns the public key as one authorized_keys line, for example "ssh-ed25519 AAAA... scimitar".
func AuthorizedKey(signer ssh.Signer) string {
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey()))) + " " + comment
}
