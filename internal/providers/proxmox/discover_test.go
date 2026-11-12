package proxmox

import (
	"context"
	"reflect"
	"testing"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

func TestDiscoverDevices(t *testing.T) {
	p := newTestProvider(t, newFixtureServer(t))
	d, err := p.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Device{
		{ID: "proxmox/node/pve1", Name: "pve1", Kind: domain.DeviceHost, Provider: "proxmox", Status: domain.StatusUp},
		{ID: "proxmox/node/pve2", Name: "pve2", Kind: domain.DeviceHost, Provider: "proxmox", Status: domain.StatusDown},
		{ID: "proxmox/guest/101", Name: "immich", Kind: domain.DeviceVM, Provider: "proxmox", Status: domain.StatusUp},
		{ID: "proxmox/guest/102", Name: "nextcloud-old", Kind: domain.DeviceVM, Provider: "proxmox", Status: domain.StatusDown},
		{ID: "proxmox/guest/200", Name: "ct-200", Kind: domain.DeviceContainer, Provider: "proxmox", Status: domain.StatusUp},
		{ID: "proxmox/guest/201", Name: "jellyfin", Kind: domain.DeviceContainer, Provider: "proxmox", Status: domain.StatusUnknown},
	}
	if !reflect.DeepEqual(d.Devices, want) {
		t.Fatalf("devices:\n got %+v\nwant %+v", d.Devices, want)
	}
}

func TestDiscoverLinksGuestsToNodes(t *testing.T) {
	p := newTestProvider(t, newFixtureServer(t))
	d, err := p.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Link{
		{From: "proxmox/guest/101", To: "proxmox/node/pve1", Kind: domain.LinkRunsOn},
		{From: "proxmox/guest/102", To: "proxmox/node/pve1", Kind: domain.LinkRunsOn},
		{From: "proxmox/guest/200", To: "proxmox/node/pve1", Kind: domain.LinkRunsOn},
		{From: "proxmox/guest/201", To: "proxmox/node/pve2", Kind: domain.LinkRunsOn},
	}
	if !reflect.DeepEqual(d.Links, want) {
		t.Fatalf("links:\n got %+v\nwant %+v", d.Links, want)
	}
}

func TestDiscoverHints(t *testing.T) {
	p := newTestProvider(t, newFixtureServer(t))
	d, err := p.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.ServiceHint{
		{DeviceID: "proxmox/guest/101", Source: "name", Value: "immich"},
		{DeviceID: "proxmox/guest/101", Source: "tag", Value: "photos"},
		{DeviceID: "proxmox/guest/101", Source: "notes", Value: "Photo library running Immich\n"},
		{DeviceID: "proxmox/guest/102", Source: "name", Value: "nextcloud-old"},
		{DeviceID: "proxmox/guest/200", Source: "name", Value: "ct-200"},
		{DeviceID: "proxmox/guest/200", Source: "tag", Value: "vaultwarden"},
		{DeviceID: "proxmox/guest/200", Source: "tag", Value: "prod"},
		{DeviceID: "proxmox/guest/200", Source: "notes", Value: "Password manager"},
		{DeviceID: "proxmox/guest/201", Source: "name", Value: "jellyfin"},
	}
	if !reflect.DeepEqual(d.Hints, want) {
		t.Fatalf("hints:\n got %+v\nwant %+v", d.Hints, want)
	}
}

func TestDiscoverUsesConfiguredName(t *testing.T) {
	srv := newFixtureServer(t)
	p, err := New(Config{Name: "lab", URL: srv.URL, TokenID: testTokenID, TokenSecret: testTokenSecret, Fingerprint: fingerprintOf(srv)})
	if err != nil {
		t.Fatal(err)
	}
	d, err := p.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "lab" || d.Devices[0].ID != "lab/node/pve1" || d.Devices[0].Provider != "lab" {
		t.Fatalf("name not applied: %s %+v", p.Name(), d.Devices[0])
	}
}
