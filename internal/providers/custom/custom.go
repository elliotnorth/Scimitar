// Package custom holds devices the user adds by hand: anything with an address
// that Scimitar has no deeper integration for, like a NAS, printer or access point.
// It only checks whether each one answers on the network.
package custom

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/elliotnorth/Scimitar/internal/domain"
	"github.com/elliotnorth/Scimitar/internal/jsonfile"
	"github.com/elliotnorth/Scimitar/internal/probe"
)

// Name is the provider name, and the prefix of every device ID it owns.
const Name = "custom"

const (
	maxDevices   = 500
	maxParallel  = 16
	maxNameBytes = 64
)

var (
	ErrNotFound = errors.New("no such device")
	ErrInvalid  = errors.New("invalid device")
)

type Device struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Kind    domain.DeviceKind `json:"kind"`
	Address string            `json:"address"`
	// Port is the TCP port that must be open for the device to count as up.
	// Zero means any sign of life on the common ports is enough.
	Port int `json:"port,omitempty"`
}

var _ domain.Provider = (*Provider)(nil)

type Provider struct {
	path string
	// check is replaced in tests.
	check func(ctx context.Context, d Device) bool

	mu      sync.Mutex
	devices []Device
}

func Open(path string) (*Provider, error) {
	p := &Provider{path: path, check: answers}
	if err := jsonfile.Load(path, &p.devices); err != nil {
		return nil, fmt.Errorf("load devices: %w", err)
	}
	return p, nil
}

func answers(ctx context.Context, d Device) bool {
	if d.Port != 0 {
		return probe.PortOpen(ctx, d.Address, d.Port)
	}
	return probe.Reachable(ctx, d.Address)
}

func (p *Provider) Name() string { return Name }

var hostname = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]{0,251}[A-Za-z0-9])?$`)

func clean(d Device) (Device, error) {
	d.Name = strings.TrimSpace(d.Name)
	d.Address = strings.TrimSpace(d.Address)
	switch {
	case d.Name == "" || len(d.Name) > maxNameBytes || strings.ContainsAny(d.Name, "\n\r\t"):
		return d, fmt.Errorf("%w: give it a name of up to %d characters", ErrInvalid, maxNameBytes)
	case !slices.Contains(domain.ManualKinds, d.Kind):
		return d, fmt.Errorf("%w: unknown kind %q", ErrInvalid, d.Kind)
	case net.ParseIP(d.Address) == nil && !hostname.MatchString(d.Address):
		return d, fmt.Errorf("%w: the address must be an IP address or a hostname, without http:// or a port", ErrInvalid)
	case d.Port < 0 || d.Port > 65535:
		return d, fmt.Errorf("%w: the port must be between 1 and 65535", ErrInvalid)
	}
	return d, nil
}

// Add saves a new device and returns it with its ID.
func (p *Provider) Add(d Device) (Device, error) {
	d, err := clean(d)
	if err != nil {
		return Device{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.devices) >= maxDevices {
		return Device{}, fmt.Errorf("%w: there are already %d devices", ErrInvalid, maxDevices)
	}
	for _, existing := range p.devices {
		if strings.EqualFold(existing.Name, d.Name) {
			return Device{}, fmt.Errorf("%w: %s is already used, pick another name", ErrInvalid, existing.Name)
		}
	}
	token := make([]byte, 6)
	if _, err := rand.Read(token); err != nil {
		return Device{}, err
	}
	d.ID = Name + "/" + hex.EncodeToString(token)
	next := append(slices.Clone(p.devices), d)
	if err := jsonfile.Save(p.path, next); err != nil {
		return Device{}, fmt.Errorf("save devices: %w", err)
	}
	p.devices = next
	return d, nil
}

// Remove deletes the device with the given ID.
func (p *Provider) Remove(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	i := slices.IndexFunc(p.devices, func(d Device) bool { return d.ID == id })
	if i < 0 {
		return ErrNotFound
	}
	next := slices.Delete(slices.Clone(p.devices), i, i+1)
	if err := jsonfile.Save(p.path, next); err != nil {
		return fmt.Errorf("save devices: %w", err)
	}
	p.devices = next
	return nil
}

func (p *Provider) list() []Device {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.devices)
}

func (p *Provider) statuses(ctx context.Context, devices []Device) map[string]domain.Status {
	out := make(map[string]domain.Status, len(devices))
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, maxParallel)
	for _, d := range devices {
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			status := domain.StatusDown
			if p.check(ctx, d) {
				status = domain.StatusUp
			}
			mu.Lock()
			out[d.ID] = status
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

func (p *Provider) Discover(ctx context.Context) (domain.Discovery, error) {
	devices := p.list()
	status := p.statuses(ctx, devices)
	var out domain.Discovery
	for _, d := range devices {
		out.Devices = append(out.Devices, domain.Device{
			ID:        d.ID,
			Name:      d.Name,
			Kind:      d.Kind,
			Provider:  Name,
			Addresses: []string{d.Address},
			Status:    status[d.ID],
		})
		out.Hints = append(out.Hints, domain.ServiceHint{DeviceID: d.ID, Source: "name", Value: d.Name})
	}
	return out, nil
}

func (p *Provider) Status(ctx context.Context) (map[string]domain.Status, error) {
	return p.statuses(ctx, p.list()), nil
}

func (p *Provider) Details(_ context.Context, deviceID string) ([]domain.Detail, error) {
	for _, d := range p.list() {
		if d.ID != deviceID {
			continue
		}
		checked := "answers on the network"
		if d.Port != 0 {
			checked = "port " + strconv.Itoa(d.Port) + " is open"
		}
		return []domain.Detail{
			{Label: "Address", Value: d.Address},
			{Label: "Up when", Value: checked},
			{Label: "Added", Value: "by hand"},
		}, nil
	}
	return nil, ErrNotFound
}
