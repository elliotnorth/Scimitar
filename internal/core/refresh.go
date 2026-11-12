package core

import (
	"context"
	"time"
)

func (inv *Inventory) Discover(ctx context.Context) {
	for _, p := range inv.registry.Providers() {
		discovery, err := p.Discover(ctx)
		inv.mu.Lock()
		state := inv.state(p.Name())
		state.err = err
		if err == nil {
			state.discovery = discovery
			state.status = nil
		}
		inv.mu.Unlock()
	}
	inv.mu.Lock()
	inv.rebuild()
	inv.mu.Unlock()
}

func (inv *Inventory) RefreshStatus(ctx context.Context) {
	for _, p := range inv.registry.Providers() {
		status, err := p.Status(ctx)
		inv.mu.Lock()
		state := inv.state(p.Name())
		state.err = err
		if err == nil {
			state.status = status
		}
		inv.mu.Unlock()
	}
	inv.mu.Lock()
	inv.rebuild()
	inv.mu.Unlock()
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
