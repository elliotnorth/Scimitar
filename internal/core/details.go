package core

import (
	"context"
	"errors"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

var ErrUnknownDevice = errors.New("no such device")

func (inv *Inventory) Details(ctx context.Context, deviceID string) ([]domain.Detail, error) {
	providerName, found := inv.deviceProvider(deviceID)
	if !found {
		return nil, ErrUnknownDevice
	}
	for _, p := range inv.registry.Providers() {
		if p.Name() == providerName {
			return p.Details(ctx, deviceID)
		}
	}
	return nil, ErrUnknownDevice
}

func (inv *Inventory) deviceProvider(deviceID string) (string, bool) {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	for _, d := range inv.snapshot.Devices {
		if d.ID == deviceID {
			return d.Provider, true
		}
	}
	return "", false
}

func (inv *Inventory) Catalog() []CatalogEntry {
	return inv.catalog
}
