package core

import (
	"context"
	"time"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

func (inv *Inventory) Discover(ctx context.Context) {
	for _, p := range inv.registry.Providers() {
		inv.discover(ctx, p)
	}
	inv.mu.Lock()
	inv.rebuild()
	inv.mu.Unlock()
}

func (inv *Inventory) discover(ctx context.Context, p domain.Provider) error {
	discovery, err := p.Discover(ctx)
	inv.mu.Lock()
	defer inv.mu.Unlock()
	state := inv.state(p.Name())
	state.discoverErr = err
	if err == nil {
		state.discovery = discovery
		state.status = nil
		state.statusErr = nil
	}
	return err
}

// RefreshStatus updates device statuses. A provider whose last discovery
// failed is discovered again first, so a source that was unreachable at
// startup shows its devices as soon as it answers instead of at the next
// full discovery.
func (inv *Inventory) RefreshStatus(ctx context.Context) {
	for _, p := range inv.registry.Providers() {
		if inv.discoveryFailed(p.Name()) && inv.discover(ctx, p) == nil {
			continue
		}
		status, err := p.Status(ctx)
		inv.mu.Lock()
		state := inv.state(p.Name())
		state.statusErr = err
		if err == nil {
			state.status = status
		}
		inv.mu.Unlock()
	}
	inv.mu.Lock()
	inv.rebuild()
	inv.mu.Unlock()
}

func (inv *Inventory) discoveryFailed(provider string) bool {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	state, ok := inv.states[provider]
	return ok && state.discoverErr != nil
}

func (inv *Inventory) Run(ctx context.Context, discoverEvery, statusEvery time.Duration) {
	inv.Discover(ctx)
	discoverTick := time.NewTicker(discoverEvery)
	statusTick := time.NewTicker(statusEvery)
	defer discoverTick.Stop()
	defer statusTick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-discoverTick.C:
			inv.Discover(ctx)
		case <-statusTick.C:
			inv.RefreshStatus(ctx)
		}
	}
}

func (inv *Inventory) state(provider string) *providerState {
	state, ok := inv.states[provider]
	if !ok {
		state = &providerState{}
		inv.states[provider] = state
	}
	return state
}
