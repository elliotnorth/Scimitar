package proxmox

import (
	"context"
	"reflect"
	"testing"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

func TestStatus(t *testing.T) {
	p := newTestProvider(t, newFixtureServer(t))
	got, err := p.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]domain.Status{
		"proxmox/node/pve1": domain.StatusUp,
		"proxmox/node/pve2": domain.StatusDown,
		"proxmox/guest/101": domain.StatusUp,
		"proxmox/guest/102": domain.StatusDown,
		"proxmox/guest/200": domain.StatusUp,
		"proxmox/guest/201": domain.StatusUnknown,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
