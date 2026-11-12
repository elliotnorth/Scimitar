package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusWhenNotInstalled(t *testing.T) {
	out := &bytes.Buffer{}
	printStatus(out, "missing.json", installPaths{unit: filepath.Join(t.TempDir(), "none.service")})
	if !strings.Contains(out.String(), "not installed") {
		t.Fatalf("got %s", out)
	}
}
