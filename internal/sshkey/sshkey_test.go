package sshkey

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOrCreateMakesAPrivateKeyOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ssh_key")
	first, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("key mode %v, want 0600", info.Mode().Perm())
	}
	second, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}
	if AuthorizedKey(first) != AuthorizedKey(second) {
		t.Fatal("second call made a new key")
	}
	if line := AuthorizedKey(first); !strings.HasPrefix(line, "ssh-ed25519 ") || !strings.HasSuffix(line, " scimitar") {
		t.Fatalf("unexpected authorized key line %q", line)
	}
}

func TestLoadDoesNotCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	if _, err := Load(path); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("Load created a key")
	}
}
