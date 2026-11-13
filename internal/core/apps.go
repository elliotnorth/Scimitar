package core

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/elliotnorth/Scimitar/internal/domain"
	"github.com/elliotnorth/Scimitar/internal/jsonfile"
	"github.com/elliotnorth/Scimitar/internal/probe"
)

const maxParallelChecks = 16

var (
	ErrUnknownService = errors.New("no such app")
	ErrInvalidAddress = errors.New("invalid address")
)

// Prober reports whether an app address answers.
type Prober interface {
	Check(ctx context.Context, target string) bool
}

// AppURLStore keeps the addresses the user set for apps, by service ID.
type AppURLStore interface {
	Load() (map[string]string, error)
	Save(map[string]string) error
}

type FileAppURLs struct {
	Path string
}

func (f FileAppURLs) Load() (map[string]string, error) {
	urls := map[string]string{}
	if err := jsonfile.Load(f.Path, &urls); err != nil {
		return nil, err
	}
	if urls == nil {
		urls = map[string]string{}
	}
	return urls, nil
}

func (f FileAppURLs) Save(urls map[string]string) error {
	return jsonfile.Save(f.Path, urls)
}

// appURL returns where a service is checked: the address the user set, or the
// app's usual port on its device when the device address is known.
func (inv *Inventory) appURL(svc domain.Service, deviceAddress string) (url string, set bool) {
	if url, ok := inv.appURLs[svc.ID]; ok {
		return url, true
	}
	entry, ok := findEntry(inv.catalog, svc.CatalogID)
	if !ok || entry.Port == 0 || deviceAddress == "" {
		return "", false
	}
	scheme := entry.Scheme
	if scheme == "" {
		scheme = "http"
	}
	return scheme + "://" + net.JoinHostPort(deviceAddress, strconv.Itoa(entry.Port)), false
}

// SetAppURL sets the address used to check and open an app. An empty address
// goes back to the automatic one.
func (inv *Inventory) SetAppURL(serviceID, address string) error {
	target := ""
	if strings.TrimSpace(address) != "" {
		var err error
		if target, err = probe.NormalizeTarget(address); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAddress, err)
		}
	}
	inv.mu.Lock()
	defer inv.mu.Unlock()
	known := false
	for _, svc := range inv.snapshot.Services {
		known = known || svc.ID == serviceID
	}
	if !known {
		return ErrUnknownService
	}
	next := maps.Clone(inv.appURLs)
	if target == "" {
		delete(next, serviceID)
	} else {
		next[serviceID] = target
	}
	if inv.urlStore != nil {
		if err := inv.urlStore.Save(next); err != nil {
			return fmt.Errorf("save app addresses: %w", err)
		}
	}
	inv.appURLs = next
	inv.rebuild()
	return nil
}

// CheckApps checks every app address now instead of waiting for the next refresh.
func (inv *Inventory) CheckApps(ctx context.Context) {
	inv.settle(ctx)
}

// settle finishes a refresh: it rebuilds the snapshot from what the providers
// reported, checks the apps, then records and announces anything that changed.
func (inv *Inventory) settle(ctx context.Context) {
	inv.mu.Lock()
	inv.rebuild()
	targets := inv.checkTargets()
	inv.mu.Unlock()

	var results map[string]domain.Status
	if inv.prober != nil && len(targets) > 0 {
		results = checkAll(ctx, inv.prober, targets)
	}

	inv.mu.Lock()
	if results != nil {
		inv.appStatus = results
		inv.rebuild()
	}
	changes := inv.detectChanges()
	var history []Event
	if len(changes) > 0 {
		inv.events = append(inv.events, changes...)
		if extra := len(inv.events) - maxStoredEvents; extra > 0 {
			inv.events = append([]Event(nil), inv.events[extra:]...)
		}
		inv.snapshot.Events = recentEvents(inv.events)
		history = append(history, inv.events...)
	}
	inv.mu.Unlock()

	if len(changes) == 0 {
		return
	}
	inv.saveEvents(history)
	if inv.notify != nil {
		inv.notify(changes)
	}
}

// checkTargets lists the app addresses worth checking. Apps on a device that
// is down are skipped: they are down too and checking them only wastes time.
func (inv *Inventory) checkTargets() []string {
	down := map[string]bool{}
	for _, d := range inv.snapshot.Devices {
		down[d.ID] = d.Status == domain.StatusDown
	}
	seen := map[string]bool{}
	var targets []string
	for _, svc := range inv.snapshot.Services {
		if svc.URL == "" || down[svc.DeviceID] || seen[svc.URL] {
			continue
		}
		seen[svc.URL] = true
		targets = append(targets, svc.URL)
	}
	return targets
}

func checkAll(ctx context.Context, prober Prober, targets []string) map[string]domain.Status {
	results := make(map[string]domain.Status, len(targets))
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, maxParallelChecks)
	for _, target := range targets {
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			status := domain.StatusDown
			if prober.Check(ctx, target) {
				status = domain.StatusUp
			}
			mu.Lock()
			results[target] = status
			mu.Unlock()
		}()
	}
	wg.Wait()
	return results
}
