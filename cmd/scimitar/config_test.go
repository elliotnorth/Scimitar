package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scimitar.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const validConfig = `{
  "providers": [
    {"type": "proxmox", "settings": {"url": "https://pve:8006", "token_id": "scimitar@pve!scimitar", "token_secret": "s", "fingerprint": "` +
	"AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB:AB" + `"}}
  ]
}`

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := loadConfig(writeConfig(t, validConfig))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StateFile != "scimitar-decisions.json" || cfg.discoverEvery() != 5*time.Minute || cfg.statusEvery() != 30*time.Second {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	cases := map[string]string{
		"not json":     `{`,
		"no providers": `{"providers": []}`,
		"zero status":  `{"status_seconds": 0, "providers": [{"type": "proxmox"}]}`,
	}
	for name, body := range cases {
		if _, err := loadConfig(writeConfig(t, body)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	if _, err := loadConfig(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("missing file: expected error")
	}
}

func TestBuildRegistry(t *testing.T) {
	cfg, err := loadConfig(writeConfig(t, validConfig))
	if err != nil {
		t.Fatal(err)
	}
	registry, err := buildRegistry(cfg.Providers)
	if err != nil {
		t.Fatal(err)
	}
	if ps := registry.Providers(); len(ps) != 1 || ps[0].Name() != "proxmox" {
		t.Fatalf("unexpected providers: %v", ps)
	}
}

func TestBuildRegistryErrors(t *testing.T) {
	cases := map[string][]providerConfig{
		"unknown type":   {{Type: "vmware"}},
		"bad settings":   {{Type: "proxmox", Settings: []byte(`{"url": "https://pve"}`)}},
		"duplicate name": {proxmoxSettings("a"), proxmoxSettings("a")},
	}
	for name, configs := range cases {
		if _, err := buildRegistry(configs); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func proxmoxSettings(name string) providerConfig {
	return providerConfig{
		Type:     "proxmox",
		Settings: []byte(`{"name": "` + name + `", "url": "https://pve", "token_id": "t", "token_secret": "s"}`),
	}
}
