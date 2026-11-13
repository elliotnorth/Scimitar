package core

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFileDecisionsMissingFileIsEmpty(t *testing.T) {
	store := FileDecisions{Path: filepath.Join(t.TempDir(), "decisions.json")}
	got, err := store.Load()
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestFileDecisionsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := FileDecisions{Path: filepath.Join(dir, "decisions.json")}
	want := Decisions{"a|name|x": "nextcloud", "b|tag|y": NotAService}
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temp files left behind: %v", entries)
	}
}

func TestFileDecisionsRejectsCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "decisions.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (FileDecisions{Path: path}).Load(); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestFileDecisionsNullFileIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "decisions.json")
	if err := os.WriteFile(path, []byte("null"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := FileDecisions{Path: path}.Load()
	if err != nil || got == nil {
		t.Fatalf("got %v, %v; want an empty map that can be written to", got, err)
	}
}
