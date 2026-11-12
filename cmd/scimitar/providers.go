package main

import (
	"encoding/json"
	"fmt"

	"github.com/elliotnorth/Scimitar/internal/core"
	"github.com/elliotnorth/Scimitar/internal/domain"
	"github.com/elliotnorth/Scimitar/internal/providers/junos"
	"github.com/elliotnorth/Scimitar/internal/providers/proxmox"
)

type providerFactory func(settings json.RawMessage, data dataDir) (domain.Provider, error)

var providerFactories = map[string]providerFactory{
	"proxmox": newProxmox,
	"junos":   newJunos,
}

func newProxmox(settings json.RawMessage, _ dataDir) (domain.Provider, error) {
	var cfg proxmox.Config
	if err := json.Unmarshal(settings, &cfg); err != nil {
		return nil, err
	}
	return proxmox.New(cfg)
}

func newJunos(settings json.RawMessage, data dataDir) (domain.Provider, error) {
	var cfg junos.Config
	if err := json.Unmarshal(settings, &cfg); err != nil {
		return nil, err
	}
	if cfg.KeyFile == "" {
		cfg.KeyFile = data.sshKey()
	}
	return junos.New(cfg)
}

func buildRegistry(configs []providerConfig, data dataDir) (*core.Registry, error) {
	var registry core.Registry
	for i, pc := range configs {
		factory, ok := providerFactories[pc.Type]
		if !ok {
			return nil, fmt.Errorf("provider %d: unknown type %q", i, pc.Type)
		}
		p, err := factory(pc.Settings, data)
		if err != nil {
			return nil, fmt.Errorf("provider %d: %w", i, err)
		}
		if err := registry.Register(p); err != nil {
			return nil, err
		}
	}
	return &registry, nil
}

// providerName returns the name a configured provider registers under.
func providerName(pc providerConfig) string {
	var named struct {
		Name string `json:"name"`
	}
	json.Unmarshal(pc.Settings, &named)
	if named.Name != "" {
		return named.Name
	}
	return pc.Type
}
