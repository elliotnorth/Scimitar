package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type providerConfig struct {
	Type     string          `json:"type"`
	Settings json.RawMessage `json:"settings"`
}

type config struct {
	StateFile       string           `json:"state_file"`
	DiscoverSeconds int              `json:"discover_seconds"`
	StatusSeconds   int              `json:"status_seconds"`
	Providers       []providerConfig `json:"providers"`
}

func loadConfig(path string) (config, error) {
	cfg := config{
		StateFile:       "scimitar-decisions.json",
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
	if len(cfg.Providers) == 0 {
		return cfg, errors.New("config has no providers")
	}
	if cfg.DiscoverSeconds <= 0 || cfg.StatusSeconds <= 0 {
		return cfg, errors.New("discover_seconds and status_seconds must be positive")
	}
	return cfg, nil
}

func (c config) discoverEvery() time.Duration {
	return time.Duration(c.DiscoverSeconds) * time.Second
}

func (c config) statusEvery() time.Duration {
	return time.Duration(c.StatusSeconds) * time.Second
}
