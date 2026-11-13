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
	Answered  []Answer
	Events    []Event
	Providers []string
	Errors    map[string]string
	UpdatedAt time.Time
}

type providerState struct {
	discovery   domain.Discovery
	status      map[string]domain.Status
	discoverErr error
	statusErr   error
}

// err is what the provider's source reports as wrong right now, if anything.
func (s *providerState) err() error {
	if s.statusErr != nil {
		return s.statusErr
	}
	return s.discoverErr
}

type Inventory struct {
	registry *Registry
	catalog  []CatalogEntry
	store    DecisionStore
	now      func() time.Time

	prober     Prober
	urlStore   AppURLStore
	eventStore EventStore
	notify     func([]Event)

	mu        sync.RWMutex
	states    map[string]*providerState
	decisions Decisions
	snapshot  Snapshot
	appURLs   map[string]string
	appStatus map[string]domain.Status
	lastKnown map[string]domain.Status
	events    []Event
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
		appURLs:   map[string]string{},
		appStatus: map[string]domain.Status{},
		lastKnown: map[string]domain.Status{},
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

// Undo forgets a decision, which puts its item back in the review queue.
func (inv *Inventory) Undo(reviewKey string) error {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	if _, decided := inv.decisions[reviewKey]; !decided {
		return ErrUnknownReviewItem
	}
	next := maps.Clone(inv.decisions)
	delete(next, reviewKey)
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
