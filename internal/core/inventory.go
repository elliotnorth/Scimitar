package core

import (
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

type Snapshot struct {
	Devices   []domain.Device
	Services  []domain.Service
	Links     []domain.Link
	Review    []ReviewItem
	Errors    map[string]string
	UpdatedAt time.Time
}

type providerState struct {
	discovery domain.Discovery
	status    map[string]domain.Status
	err       error
}

type Inventory struct {
	registry *Registry
	catalog  []CatalogEntry
	store    DecisionStore
	now      func() time.Time

	mu        sync.RWMutex
	states    map[string]*providerState
	decisions Decisions
	snapshot  Snapshot
}

func NewInventory(registry *Registry, catalog []CatalogEntry, store DecisionStore) (*Inventory, error) {
	decisions, err := store.Load()
	if err != nil {
		return nil, fmt.Errorf("load decisions: %w", err)
	}
	return &Inventory{
		registry:  registry,
		catalog:   catalog,
		store:     store,
		now:       time.Now,
		states:    map[string]*providerState{},
		decisions: decisions,
	}, nil
}

func (inv *Inventory) Snapshot() Snapshot {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.snapshot
}

func (inv *Inventory) Decide(reviewKey, catalogID string) error {
	if err := validateDecision(inv.catalog, catalogID); err != nil {
		return err
	}
	inv.mu.Lock()
	defer inv.mu.Unlock()
	if !inv.inReview(reviewKey) {
		return ErrUnknownReviewItem
	}
	next := maps.Clone(inv.decisions)
	next[reviewKey] = catalogID
	if err := inv.store.Save(next); err != nil {
		return fmt.Errorf("save decisions: %w", err)
	}
	inv.decisions = next
	inv.rebuild()
	return nil
}

func (inv *Inventory) inReview(key string) bool {
	for _, item := range inv.snapshot.Review {
		if item.Key() == key {
			return true
		}
	}
	return false
}
