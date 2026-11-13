package core

import (
	"context"
	"errors"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

var errUnreachable = errors.New("unreachable")

type fakeProvider struct {
	name      string
	discovery domain.Discovery
	status    map[string]domain.Status
	err       error
	// discoverErr fails discovery only, while status keeps working.
	discoverErr error
	discovers   int
}

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) Discover(context.Context) (domain.Discovery, error) {
	f.discovers++
	if f.discoverErr != nil {
		return domain.Discovery{}, f.discoverErr
	}
	return f.discovery, f.err
}

func (f *fakeProvider) Status(context.Context) (map[string]domain.Status, error) {
	return f.status, f.err
}

func (f *fakeProvider) Details(context.Context, string) ([]domain.Detail, error) {
	return nil, f.err
}

type memoryDecisions struct {
	saved   Decisions
	saveErr error
}

func (m *memoryDecisions) Load() (Decisions, error) {
	if m.saved == nil {
		return Decisions{}, nil
	}
	return m.saved, nil
}

func (m *memoryDecisions) Save(d Decisions) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = d
	return nil
}

func labDiscovery() domain.Discovery {
	return domain.Discovery{
		Devices: []domain.Device{
			{ID: "lab/node/pve1", Name: "pve1", Kind: domain.DeviceHost, Provider: "lab", Status: domain.StatusUp},
			{ID: "lab/guest/101", Name: "immich", Kind: domain.DeviceVM, Provider: "lab", Status: domain.StatusUp},
			{ID: "lab/guest/102", Name: "nextcloud-old", Kind: domain.DeviceVM, Provider: "lab", Status: domain.StatusDown},
		},
		Links: []domain.Link{
			{From: "lab/guest/101", To: "lab/node/pve1", Kind: domain.LinkRunsOn},
			{From: "lab/guest/102", To: "lab/node/pve1", Kind: domain.LinkRunsOn},
		},
		Hints: []domain.ServiceHint{
			{DeviceID: "lab/guest/101", Source: "name", Value: "immich"},
			{DeviceID: "lab/guest/102", Source: "name", Value: "nextcloud-old"},
		},
	}
}
