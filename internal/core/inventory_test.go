package core

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

func newLabInventory(t *testing.T, store *memoryDecisions) (*Inventory, *fakeProvider) {
	t.Helper()
	provider := &fakeProvider{name: "lab", discovery: labDiscovery()}
	var registry Registry
	if err := registry.Register(provider); err != nil {
		t.Fatal(err)
	}
	inv, err := NewInventory(&registry, DefaultCatalog, store)
	if err != nil {
		t.Fatal(err)
	}
	return inv, provider
}

func statuses(devices []domain.Device) map[string]domain.Status {
	out := map[string]domain.Status{}
	for _, d := range devices {
		out[d.ID] = d.Status
	}
	return out
}

func TestDiscoverBuildsSnapshot(t *testing.T) {
	inv, _ := newLabInventory(t, &memoryDecisions{})
	inv.Discover(context.Background())
	s := inv.Snapshot()

	if len(s.Devices) != 3 || len(s.Links) != 2 || len(s.Errors) != 0 {
		t.Fatalf("unexpected snapshot: %+v", s)
	}
	if !reflect.DeepEqual(serviceIDs(s.Services), []string{"lab/guest/101/immich"}) {
		t.Fatalf("services: %v", serviceIDs(s.Services))
	}
	if len(s.Review) != 1 || s.Review[0].Hint.Value != "nextcloud-old" {
		t.Fatalf("review: %+v", s.Review)
	}
}

func TestServiceStatusFollowsDownDeviceOnly(t *testing.T) {
	inv, _ := newLabInventory(t, &memoryDecisions{})
	inv.Discover(context.Background())
	key := inv.Snapshot().Review[0].Key()
	if err := inv.Decide(key, "nextcloud"); err != nil {
		t.Fatal(err)
	}
	got := map[string]domain.Status{}
	for _, s := range inv.Snapshot().Services {
		got[s.ID] = s.Status
	}
	want := map[string]domain.Status{
		"lab/guest/101/immich":    domain.StatusUnknown,
		"lab/guest/102/nextcloud": domain.StatusDown,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestRefreshStatusOverridesDiscovery(t *testing.T) {
	inv, provider := newLabInventory(t, &memoryDecisions{})
	inv.Discover(context.Background())
	provider.status = map[string]domain.Status{
		"lab/node/pve1": domain.StatusUp,
		"lab/guest/101": domain.StatusDown,
	}
	inv.RefreshStatus(context.Background())
	want := map[string]domain.Status{
		"lab/node/pve1": domain.StatusUp,
		"lab/guest/101": domain.StatusDown,
		"lab/guest/102": domain.StatusUnknown,
	}
	if got := statuses(inv.Snapshot().Devices); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestProviderFailureKeepsDevicesAsUnknown(t *testing.T) {
	inv, provider := newLabInventory(t, &memoryDecisions{})
	inv.Discover(context.Background())
	provider.err = errUnreachable
	inv.RefreshStatus(context.Background())
	s := inv.Snapshot()

	if len(s.Devices) != 3 {
		t.Fatalf("devices dropped: %+v", s.Devices)
	}
	for id, status := range statuses(s.Devices) {
		if status != domain.StatusUnknown {
			t.Errorf("%s: got %s, want unknown", id, status)
		}
	}
	if s.Errors["lab"] != "unreachable" {
		t.Fatalf("errors: %v", s.Errors)
	}

	provider.err = nil
	inv.Discover(context.Background())
	if s := inv.Snapshot(); len(s.Errors) != 0 || statuses(s.Devices)["lab/guest/101"] != domain.StatusUp {
		t.Fatalf("did not recover: %+v", s)
	}
}

func TestDecideConfirmsService(t *testing.T) {
	store := &memoryDecisions{}
	inv, _ := newLabInventory(t, store)
	inv.Discover(context.Background())
	key := inv.Snapshot().Review[0].Key()

	if err := inv.Decide(key, "nextcloud"); err != nil {
		t.Fatal(err)
	}
	s := inv.Snapshot()
	if len(s.Review) != 0 {
		t.Fatalf("review not cleared: %+v", s.Review)
	}
	if !reflect.DeepEqual(serviceIDs(s.Services), []string{"lab/guest/101/immich", "lab/guest/102/nextcloud"}) {
		t.Fatalf("services: %v", serviceIDs(s.Services))
	}
	if store.saved[key] != "nextcloud" {
		t.Fatalf("decision not saved: %v", store.saved)
	}
}

func TestDecideNotAService(t *testing.T) {
	inv, _ := newLabInventory(t, &memoryDecisions{})
	inv.Discover(context.Background())
	if err := inv.Decide(inv.Snapshot().Review[0].Key(), NotAService); err != nil {
		t.Fatal(err)
	}
	s := inv.Snapshot()
	if len(s.Review) != 0 || len(s.Services) != 1 {
		t.Fatalf("got %+v", s)
	}
}

func TestDecisionsSurviveRestart(t *testing.T) {
	store := &memoryDecisions{}
	first, _ := newLabInventory(t, store)
	first.Discover(context.Background())
	if err := first.Decide(first.Snapshot().Review[0].Key(), "nextcloud"); err != nil {
		t.Fatal(err)
	}

	second, _ := newLabInventory(t, store)
	second.Discover(context.Background())
	if s := second.Snapshot(); len(s.Review) != 0 || len(s.Services) != 2 {
		t.Fatalf("decision lost after restart: %+v", s)
	}
}

func TestDecideRejectsBadInput(t *testing.T) {
	inv, _ := newLabInventory(t, &memoryDecisions{})
	inv.Discover(context.Background())
	key := inv.Snapshot().Review[0].Key()

	if err := inv.Decide("lab/guest/999|name|x", "nextcloud"); !errors.Is(err, ErrUnknownReviewItem) {
		t.Errorf("unknown key: got %v", err)
	}
	if err := inv.Decide(key, "not-in-catalog"); err == nil {
		t.Error("unknown catalog id accepted")
	}
	if len(inv.Snapshot().Review) != 1 {
		t.Fatal("bad input changed the review queue")
	}
}

func TestDecideKeepsStateWhenSaveFails(t *testing.T) {
	store := &memoryDecisions{saveErr: errors.New("disk full")}
	inv, _ := newLabInventory(t, store)
	inv.Discover(context.Background())
	if err := inv.Decide(inv.Snapshot().Review[0].Key(), "nextcloud"); err == nil {
		t.Fatal("expected save error")
	}
	if len(inv.Snapshot().Review) != 1 {
		t.Fatal("decision applied although it was not saved")
	}
}
