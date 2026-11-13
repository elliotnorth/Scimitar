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

	if len(s.Devices) != 3 || len(s.Links) != 2 || len(s.Errors) != 0 || !reflect.DeepEqual(s.Providers, []string{"lab"}) {
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

func TestDetailsRoutesToOwningProvider(t *testing.T) {
	inv, provider := newLabInventory(t, &memoryDecisions{})
	inv.Discover(context.Background())
	if _, err := inv.Details(context.Background(), "lab/guest/101"); err != nil {
		t.Fatal(err)
	}
	provider.err = errUnreachable
	if _, err := inv.Details(context.Background(), "lab/guest/101"); !errors.Is(err, errUnreachable) {
		t.Fatalf("provider error not passed through: %v", err)
	}
	if _, err := inv.Details(context.Background(), "other/guest/1"); !errors.Is(err, ErrUnknownDevice) {
		t.Fatalf("got %v", err)
	}
}

func TestFailedDiscoveryIsRetriedOnStatusRefresh(t *testing.T) {
	inv, provider := newLabInventory(t, &memoryDecisions{})
	provider.discoverErr = errUnreachable
	inv.Discover(context.Background())
	if s := inv.Snapshot(); len(s.Devices) != 0 || s.Errors["lab"] != "unreachable" {
		t.Fatalf("after failed discovery: %+v", s)
	}

	// Status works but discovery still fails: the error must stay visible.
	inv.RefreshStatus(context.Background())
	if s := inv.Snapshot(); s.Errors["lab"] != "unreachable" {
		t.Fatalf("discovery error hidden by a working status check: %+v", s)
	}

	provider.discoverErr = nil
	inv.RefreshStatus(context.Background())
	if s := inv.Snapshot(); len(s.Devices) != 3 || len(s.Errors) != 0 {
		t.Fatalf("devices did not appear once the source answered: %+v", s)
	}

	before := provider.discovers
	inv.RefreshStatus(context.Background())
	if provider.discovers != before {
		t.Fatal("status refresh ran discovery although the last one worked")
	}
}

func TestUndoPutsItemBackInReview(t *testing.T) {
	store := &memoryDecisions{}
	inv, _ := newLabInventory(t, store)
	inv.Discover(context.Background())
	key := inv.Snapshot().Review[0].Key()
	if err := inv.Decide(key, NotAService); err != nil {
		t.Fatal(err)
	}
	s := inv.Snapshot()
	if len(s.Answered) != 1 || s.Answered[0].Key() != key || s.Answered[0].CatalogID != NotAService {
		t.Fatalf("answered: %+v", s.Answered)
	}

	if err := inv.Undo(key); err != nil {
		t.Fatal(err)
	}
	s = inv.Snapshot()
	if len(s.Review) != 1 || len(s.Answered) != 0 {
		t.Fatalf("after undo: %+v", s)
	}
	if _, kept := store.saved[key]; kept {
		t.Fatal("undo was not saved")
	}
	if err := inv.Undo(key); !errors.Is(err, ErrUnknownReviewItem) {
		t.Fatalf("second undo: got %v", err)
	}
}

func TestUndoKeepsStateWhenSaveFails(t *testing.T) {
	store := &memoryDecisions{}
	inv, _ := newLabInventory(t, store)
	inv.Discover(context.Background())
	key := inv.Snapshot().Review[0].Key()
	if err := inv.Decide(key, "nextcloud"); err != nil {
		t.Fatal(err)
	}
	store.saveErr = errors.New("disk full")
	if err := inv.Undo(key); err == nil {
		t.Fatal("expected save error")
	}
	if s := inv.Snapshot(); len(s.Answered) != 1 || len(s.Services) != 2 {
		t.Fatalf("undo applied although it was not saved: %+v", s)
	}
}

type fakeProber struct {
	down map[string]bool
	seen []string
}

func (f *fakeProber) Check(_ context.Context, target string) bool {
	f.seen = append(f.seen, target)
	return !f.down[target]
}

type memoryURLs struct{ saved map[string]string }

func (m *memoryURLs) Load() (map[string]string, error) { return map[string]string{}, nil }
func (m *memoryURLs) Save(urls map[string]string) error {
	m.saved = urls
	return nil
}

type memoryEvents struct{ saved []Event }

func (m *memoryEvents) Load() ([]Event, error) { return nil, nil }
func (m *memoryEvents) Save(events []Event) error {
	m.saved = events
	return nil
}

func serviceByID(t *testing.T, inv *Inventory, id string) domain.Service {
	t.Helper()
	for _, s := range inv.Snapshot().Services {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no service %s", id)
	return domain.Service{}
}

func TestAppIsCheckedAtItsAddress(t *testing.T) {
	inv, provider := newLabInventory(t, &memoryDecisions{})
	prober := &fakeProber{down: map[string]bool{}}
	urls := &memoryURLs{}
	if err := inv.Configure(Extras{Prober: prober, AppURLs: urls}); err != nil {
		t.Fatal(err)
	}
	inv.Discover(context.Background())
	const immich = "lab/guest/101/immich"

	// No address known for the device and none set: nothing to check.
	if s := serviceByID(t, inv, immich); s.Status != domain.StatusUnknown || s.URL != "" || len(prober.seen) != 0 {
		t.Fatalf("unchecked app: %+v, probed %v", s, prober.seen)
	}

	// A device address gives the app its usual port.
	provider.discovery.Devices[1].Addresses = []string{"10.0.0.5"}
	inv.Discover(context.Background())
	if s := serviceByID(t, inv, immich); s.Status != domain.StatusUp || s.URL != "http://10.0.0.5:2283" || s.URLSet {
		t.Fatalf("automatic address: %+v", s)
	}

	// The user's own address wins and is saved.
	if err := inv.SetAppURL(immich, "photos.lan"); err != nil {
		t.Fatal(err)
	}
	prober.down["http://photos.lan"] = true
	inv.CheckApps(context.Background())
	if s := serviceByID(t, inv, immich); s.Status != domain.StatusDown || s.URL != "http://photos.lan" || !s.URLSet {
		t.Fatalf("own address: %+v", s)
	}
	if urls.saved[immich] != "http://photos.lan" {
		t.Fatalf("address not saved: %v", urls.saved)
	}

	// Clearing it goes back to the automatic one.
	if err := inv.SetAppURL(immich, " "); err != nil {
		t.Fatal(err)
	}
	inv.CheckApps(context.Background())
	if s := serviceByID(t, inv, immich); s.Status != domain.StatusUp || s.URLSet {
		t.Fatalf("after clearing: %+v", s)
	}

	if err := inv.SetAppURL(immich, "ftp://x"); !errors.Is(err, ErrInvalidAddress) {
		t.Errorf("bad address: got %v", err)
	}
	if err := inv.SetAppURL("lab/guest/999/immich", "x.lan"); !errors.Is(err, ErrUnknownService) {
		t.Errorf("unknown app: got %v", err)
	}
}

func TestAppOnDownDeviceIsNotChecked(t *testing.T) {
	inv, provider := newLabInventory(t, &memoryDecisions{})
	prober := &fakeProber{}
	inv.Configure(Extras{Prober: prober})
	provider.discovery.Devices[1].Addresses = []string{"10.0.0.5"}
	provider.discovery.Devices[1].Status = domain.StatusDown
	inv.Discover(context.Background())
	if s := serviceByID(t, inv, "lab/guest/101/immich"); s.Status != domain.StatusDown || len(prober.seen) != 0 {
		t.Fatalf("got %+v, probed %v", s, prober.seen)
	}
}

func TestEventsRecordChangesOnly(t *testing.T) {
	inv, provider := newLabInventory(t, &memoryDecisions{})
	prober := &fakeProber{down: map[string]bool{}}
	store := &memoryEvents{}
	var notified []Event
	inv.Configure(Extras{Prober: prober, Events: store, Notify: func(e []Event) { notified = append(notified, e...) }})
	provider.discovery.Devices[1].Addresses = []string{"10.0.0.5"}
	ctx := context.Background()

	inv.Discover(ctx)
	inv.RefreshStatus(ctx)
	if len(notified) != 0 || len(inv.Snapshot().Events) != 0 {
		t.Fatalf("first sighting made events: %+v", notified)
	}

	// The app stops answering while its VM stays up.
	prober.down["http://10.0.0.5:2283"] = true
	inv.RefreshStatus(ctx)
	if len(notified) != 1 || notified[0].Subject != SubjectApp || notified[0].Status != domain.StatusDown || notified[0].Name != "Immich" {
		t.Fatalf("app down: %+v", notified)
	}

	// The VM goes down: one device event, and no second event for its app.
	provider.status = map[string]domain.Status{"lab/node/pve1": domain.StatusUp, "lab/guest/101": domain.StatusDown, "lab/guest/102": domain.StatusDown}
	inv.RefreshStatus(ctx)
	if len(notified) != 2 || notified[1].Subject != SubjectDevice || notified[1].ID != "lab/guest/101" || notified[1].Status != domain.StatusDown {
		t.Fatalf("device down: %+v", notified)
	}

	// The source fails, then everything comes back.
	provider.err = errUnreachable
	inv.RefreshStatus(ctx)
	if len(notified) != 3 || notified[2].Subject != SubjectSource || notified[2].Detail != "unreachable" {
		t.Fatalf("source down: %+v", notified)
	}
	provider.err = nil
	provider.status["lab/guest/101"] = domain.StatusUp
	delete(prober.down, "http://10.0.0.5:2283")
	inv.RefreshStatus(ctx)
	if len(notified) != 6 {
		t.Fatalf("recovery: %+v", notified[3:])
	}
	for _, e := range notified[3:] {
		if e.Status != domain.StatusUp {
			t.Errorf("recovery event is not up: %+v", e)
		}
	}

	shown := inv.Snapshot().Events
	if len(shown) != 6 || len(store.saved) != 6 || shown[0].Status != domain.StatusUp || shown[5].Subject != SubjectApp {
		t.Fatalf("history: shown %d, saved %d", len(shown), len(store.saved))
	}

	// Nothing changed: nothing new.
	inv.RefreshStatus(ctx)
	if len(notified) != 6 {
		t.Fatal("event without a change")
	}
}
