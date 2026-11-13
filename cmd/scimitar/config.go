package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type providerConfig struct {
	Type     string          `json:"type"`
	Settings json.RawMessage `json:"settings"`
}

type config struct {
	Listen          string           `json:"listen"`
	DataDir         dataDir          `json:"data_dir"`
	DiscoverSeconds int              `json:"discover_seconds"`
	StatusSeconds   int              `json:"status_seconds"`
	Providers       []providerConfig `json:"providers"`
}

func loadConfig(path string) (config, error) {
	cfg := config{
		Listen:          "127.0.0.1:8080",
		DataDir:         "scimitar-data",
		DiscoverSeconds: 300,
		StatusSeconds:   30,
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.DiscoverSeconds <= 0 || cfg.StatusSeconds <= 0 {
		return cfg, errors.New("discover_seconds and status_seconds must be positive")
	}
	return cfg, nil
}

func (c config) hasProvider(kind string) bool {
	for _, p := range c.Providers {
		if p.Type == kind {
			return true
		}
	}
	return false
}

// saveConfig writes the config readable by root only, replacing the old file in one step.
func saveConfig(path string, cfg config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".scimitar-config-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func (c config) discoverEvery() time.Duration {
	return time.Duration(c.DiscoverSeconds) * time.Second
}

func (c config) statusEvery() time.Duration {
	return time.Duration(c.StatusSeconds) * time.Second
}
