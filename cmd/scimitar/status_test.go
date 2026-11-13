package main

import (
	"bytes"
	"encoding/json"
	"os"
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

func TestStatusShowsWhereEachSourceIs(t *testing.T) {
	dir := t.TempDir()
	paths := installPaths{unit: filepath.Join(dir, "scimitar.service"), configDir: dir}
	os.WriteFile(paths.unit, []byte("unit"), 0o644)
	cfg := config{Listen: "127.0.0.1:8080", DataDir: dataDir(dir), DiscoverSeconds: 300, StatusSeconds: 30, Providers: []providerConfig{
		{Type: "proxmox", Settings: json.RawMessage(`{"url":"https://pve:8006"}`)},
		{Type: "junos", Settings: json.RawMessage(`{"name":"core-sw","address":"10.0.0.2:22"}`)},
	}}
	if err := saveConfig(paths.configFile(), cfg); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	printStatus(out, paths.configFile(), paths)
	for _, want := range []string{"proxmox at https://pve:8006", "junos at 10.0.0.2:22"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
