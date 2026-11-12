package proxmox

import (
	"context"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFingerprintAcceptsColonSeparatedForm(t *testing.T) {
	srv := newFixtureServer(t)
	raw := fingerprintOf(srv)
	var pairs []string
	for i := 0; i < len(raw); i += 2 {
		pairs = append(pairs, raw[i:i+2])
	}
	p, err := New(Config{URL: srv.URL, TokenID: testTokenID, TokenSecret: testTokenSecret, Fingerprint: strings.Join(pairs, ":")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Status(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestFingerprintMismatchFails(t *testing.T) {
	srv := newFixtureServer(t)
	p, err := New(Config{URL: srv.URL, TokenID: testTokenID, TokenSecret: testTokenSecret, Fingerprint: strings.Repeat("AB", 32)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Status(context.Background()); err == nil || !strings.Contains(err.Error(), "fingerprint mismatch") {
		t.Fatalf("expected fingerprint mismatch, got %v", err)
	}
}

func TestUnpinnedSelfSignedFails(t *testing.T) {
	srv := newFixtureServer(t)
	p, err := New(Config{URL: srv.URL, TokenID: testTokenID, TokenSecret: testTokenSecret})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Status(context.Background()); err == nil {
		t.Fatal("expected certificate verification to fail")
	}
}

func TestCAFileTrustsServer(t *testing.T) {
	srv := newFixtureServer(t)
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(caPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := New(Config{URL: srv.URL, TokenID: testTokenID, TokenSecret: testTokenSecret, CAFile: caPath})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Status(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestWrongTokenFails(t *testing.T) {
	srv := newFixtureServer(t)
	p, err := New(Config{URL: srv.URL, TokenID: testTokenID, TokenSecret: "wrong", Fingerprint: fingerprintOf(srv)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Status(context.Background()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401, got %v", err)
	}
}

func TestConfigValidation(t *testing.T) {
	cases := []Config{
		{TokenID: "a", TokenSecret: "b"},
		{URL: "https://pve:8006"},
		{URL: "https://pve:8006", TokenID: "a", TokenSecret: "b", Fingerprint: "AB", CAFile: "ca.pem"},
		{URL: "https://pve:8006", TokenID: "a", TokenSecret: "b", Fingerprint: "not-hex"},
		{URL: "https://pve:8006", TokenID: "a", TokenSecret: "b", CAFile: "/nonexistent/ca.pem"},
	}
	for i, cfg := range cases {
		if _, err := New(cfg); err == nil {
			t.Errorf("case %d: expected error", i)
		}
	}
}
