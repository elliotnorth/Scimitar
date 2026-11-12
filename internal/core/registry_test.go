package core

import (
	"context"
	"testing"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

type stubProvider struct{ name string }

func (s stubProvider) Name() string { return s.name }

func (stubProvider) Discover(context.Context) (domain.Discovery, error) {
	return domain.Discovery{}, nil
}

func (stubProvider) Status(context.Context) (map[string]domain.Status, error) {
	return nil, nil
}

func (stubProvider) Details(context.Context, string) ([]domain.Detail, error) {
	return nil, nil
}

func TestRegisterKeepsOrder(t *testing.T) {
	var r Registry
	for _, name := range []string{"proxmox", "linux"} {
		if err := r.Register(stubProvider{name}); err != nil {
			t.Fatal(err)
		}
	}
	got := r.Providers()
	if len(got) != 2 || got[0].Name() != "proxmox" || got[1].Name() != "linux" {
		t.Fatalf("unexpected providers: %v", got)
	}
}

func TestRegisterRejectsDuplicateName(t *testing.T) {
	var r Registry
	if err := r.Register(stubProvider{"proxmox"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(stubProvider{"proxmox"}); err == nil {
		t.Fatal("expected duplicate registration to fail")
	}
}

func TestProvidersReturnsCopy(t *testing.T) {
	var r Registry
	_ = r.Register(stubProvider{"proxmox"})
	r.Providers()[0] = stubProvider{"tampered"}
	if r.Providers()[0].Name() != "proxmox" {
		t.Fatal("registry was modified through returned slice")
	}
}
