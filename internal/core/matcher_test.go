package core

import (
	"reflect"
	"testing"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

func hint(device, value string) domain.ServiceHint {
	return domain.ServiceHint{DeviceID: device, Source: "name", Value: value}
}

func serviceIDs(services []domain.Service) []string {
	var ids []string
	for _, s := range services {
		ids = append(ids, s.ID)
	}
	return ids
}

func TestExactMatchIgnoresCaseAndSeparators(t *testing.T) {
	for _, value := range []string{"immich", "Immich", "Home Assistant", "home_assistant", "PI-HOLE"} {
		got := Match(DefaultCatalog, []domain.ServiceHint{hint("vm1", value)})
		if len(got.Services) != 1 || len(got.Review) != 0 {
			t.Errorf("%q: want one confident service, got %+v", value, got)
		}
	}
}

func TestExactMatchBuildsService(t *testing.T) {
	got := Match(DefaultCatalog, []domain.ServiceHint{hint("pve/qemu/101", "bitwarden")})
	want := domain.Service{
		ID:        "pve/qemu/101/vaultwarden",
		Name:      "Vaultwarden",
		CatalogID: "vaultwarden",
		DeviceID:  "pve/qemu/101",
		Status:    domain.StatusUnknown,
	}
	if len(got.Services) != 1 || got.Services[0] != want {
		t.Fatalf("got %+v, want %+v", got.Services, want)
	}
}

func TestPartialMatchGoesToReview(t *testing.T) {
	got := Match(DefaultCatalog, []domain.ServiceHint{hint("vm1", "nextcloud-old")})
	if len(got.Services) != 0 {
		t.Fatalf("partial match must not create a service, got %+v", got.Services)
	}
	if len(got.Review) != 1 || !reflect.DeepEqual(got.Review[0].CandidateIDs, []string{"nextcloud"}) {
		t.Fatalf("unexpected review: %+v", got.Review)
	}
}

func TestSeveralCandidatesGoToReview(t *testing.T) {
	got := Match(DefaultCatalog, []domain.ServiceHint{hint("vm1", "immich-and-jellyfin")})
	if len(got.Services) != 0 || len(got.Review) != 1 {
		t.Fatalf("got %+v", got)
	}
	if !reflect.DeepEqual(got.Review[0].CandidateIDs, []string{"immich", "jellyfin"}) {
		t.Fatalf("unexpected candidates: %v", got.Review[0].CandidateIDs)
	}
}

func TestExactAliasWinsOverContainedAlias(t *testing.T) {
	got := Match(DefaultCatalog, []domain.ServiceHint{hint("vm1", "adguard-home")})
	if !reflect.DeepEqual(serviceIDs(got.Services), []string{"vm1/adguard-home"}) || len(got.Review) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestUnknownHintIsIgnored(t *testing.T) {
	got := Match(DefaultCatalog, []domain.ServiceHint{hint("vm1", "debian-12"), hint("vm2", "")})
	if len(got.Services) != 0 || len(got.Review) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestDuplicateHintsCreateOneService(t *testing.T) {
	hints := []domain.ServiceHint{
		hint("vm1", "immich"),
		{DeviceID: "vm1", Source: "tag", Value: "Immich"},
	}
	got := Match(DefaultCatalog, hints)
	if !reflect.DeepEqual(serviceIDs(got.Services), []string{"vm1/immich"}) {
		t.Fatalf("got %v", serviceIDs(got.Services))
	}
}

func TestConfirmedServiceIsDroppedFromReview(t *testing.T) {
	hints := []domain.ServiceHint{
		hint("vm1", "immich-prod"),
		{DeviceID: "vm1", Source: "tag", Value: "immich"},
	}
	got := Match(DefaultCatalog, hints)
	if len(got.Review) != 0 {
		t.Fatalf("review should be empty once confirmed, got %+v", got.Review)
	}
}

func TestSameServiceOnDifferentDevices(t *testing.T) {
	got := Match(DefaultCatalog, []domain.ServiceHint{hint("vm1", "gitea"), hint("vm2", "gitea")})
	if !reflect.DeepEqual(serviceIDs(got.Services), []string{"vm1/gitea", "vm2/gitea"}) {
		t.Fatalf("got %v", serviceIDs(got.Services))
	}
}
