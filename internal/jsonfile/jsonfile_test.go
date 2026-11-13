package jsonfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTripIsPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.json")
	got := map[string]int{"kept": 1}
	if err := Load(path, &got); err != nil || got["kept"] != 1 {
		t.Fatalf("missing file: %v %v", got, err)
	}
	if err := Save(path, map[string]int{"a": 2}); err != nil {
		t.Fatal(err)
	}
	got = nil
	if err := Load(path, &got); err != nil || got["a"] != 2 {
		t.Fatalf("got %v %v", got, err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", info.Mode())
	}
	os.WriteFile(path, []byte("{"), 0o600)
	if err := Load(path, &got); err == nil {
		t.Fatal("corrupt file accepted")
	}
}
