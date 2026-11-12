package core

import (
	"reflect"
	"testing"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

func TestResolveLinks(t *testing.T) {
	devices := []domain.Device{
		{ID: "lab/node/pve", Name: "pve"},
		{ID: "ex3400/device", Name: "ex3400"},
		{ID: "srx/device", Name: "SRX"},
	}
	links := []domain.Link{
		{From: "lab/guest/101", To: "lab/node/pve", Kind: domain.LinkRunsOn},
		{From: "ex3400/device", To: domain.NameRef("pve.home.lan"), Kind: domain.LinkConnected},
		{From: "ex3400/device", To: domain.NameRef("srx"), Kind: domain.LinkConnected},
		{From: "srx/device", To: domain.NameRef("ex3400"), Kind: domain.LinkConnected},
		{From: "ex3400/device", To: domain.NameRef("printer"), Kind: domain.LinkConnected},
		{From: "ex3400/device", To: domain.NameRef("ex3400"), Kind: domain.LinkConnected},
	}
	want := []domain.Link{
		{From: "lab/guest/101", To: "lab/node/pve", Kind: domain.LinkRunsOn},
		{From: "ex3400/device", To: "lab/node/pve", Kind: domain.LinkConnected},
		{From: "ex3400/device", To: "srx/device", Kind: domain.LinkConnected},
	}
	if got := resolveLinks(links, devices); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}
