package proxmox

import (
	"context"
	"reflect"
	"testing"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

func TestNodeDetails(t *testing.T) {
	p := newTestProvider(t, newFixtureServer(t))
	got, err := p.Details(context.Background(), "proxmox/node/pve1")
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Detail{
		{Label: "Status", Value: "online"},
		{Label: "CPUs", Value: "8"},
		{Label: "Memory", Value: "32.0 GiB"},
		{Label: "Uptime", Value: "3d 2h"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestGuestDetails(t *testing.T) {
	p := newTestProvider(t, newFixtureServer(t))
	got, err := p.Details(context.Background(), "proxmox/guest/200")
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Detail{
		{Label: "VMID", Value: "200"},
		{Label: "Type", Value: "lxc"},
		{Label: "Node", Value: "pve1"},
		{Label: "Status", Value: "running"},
		{Label: "CPUs", Value: "1"},
		{Label: "Memory", Value: "0.5 GiB"},
		{Label: "Uptime", Value: "1h 2m"},
		{Label: "Notes", Value: "Password manager"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestDetailsRejectsBadIDs(t *testing.T) {
	p := newTestProvider(t, newFixtureServer(t))
	for _, id := range []string{"other/node/pve1", "proxmox/node/", "proxmox/guest/abc", "proxmox/guest/999", "proxmox/node/pve9"} {
		if _, err := p.Details(context.Background(), id); err == nil {
			t.Errorf("%q: expected error", id)
		}
	}
}
