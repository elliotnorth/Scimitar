package proxmox

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

func tlsConfig(cfg Config) (*tls.Config, error) {
	switch {
	case cfg.Fingerprint != "":
		return pinnedTLS(cfg.Fingerprint)
	case cfg.CAFile != "":
		return caTLS(cfg.CAFile)
	}
	return &tls.Config{MinVersion: tls.VersionTLS12}, nil
}

func pinnedTLS(fingerprint string) (*tls.Config, error) {
	want, err := hex.DecodeString(strings.ReplaceAll(fingerprint, ":", ""))
	if err != nil || len(want) != sha256.Size {
		return nil, errors.New("proxmox: fingerprint must be a sha256 hex digest")
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		// Proxmox ships a self-signed certificate, so the pinned fingerprint replaces chain verification.
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errors.New("proxmox: server sent no certificate")
			}
			got := sha256.Sum256(rawCerts[0])
			if !bytes.Equal(got[:], want) {
				return fmt.Errorf("proxmox: certificate fingerprint mismatch, got %X", got)
			}
			return nil
		},
	}, nil
}

func caTLS(path string) (*tls.Config, error) {
	pem, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("proxmox: read ca file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, errors.New("proxmox: ca file contains no certificates")
	}
	return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}, nil
}
