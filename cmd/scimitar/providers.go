package main

import (
	"encoding/json"
	"fmt"

	"github.com/elliotnorth/Scimitar/internal/core"
	"github.com/elliotnorth/Scimitar/internal/domain"
	"github.com/elliotnorth/Scimitar/internal/providers/proxmox"
)

type providerFactory func(settings json.RawMessage) (domain.Provider, error)

var providerFactories = map[string]providerFactory{
	"proxmox": newProxmox,
}

func newProxmox(settings json.RawMessage) (domain.Provider, error) {
	var cfg proxmox.Config
	if err := json.Unmarshal(settings, &cfg); err != nil {
		return nil, err
	}
	return proxmox.New(cfg)
}

func buildRegistry(configs []providerConfig) (*core.Registry, error) {
	var registry core.Registry
	for i, pc := range configs {
		factory, ok := providerFactories[pc.Type]
		if !ok {
			return nil, fmt.Errorf("provider %d: unknown type %q", i, pc.Type)
		}
		p, err := factory(pc.Settings)
		if err != nil {
			return nil, fmt.Errorf("provider %d: %w", i, err)
		}
		if err := registry.Register(p); err != nil {
			return nil, err
		}
	}
	return &registry, nil
}
