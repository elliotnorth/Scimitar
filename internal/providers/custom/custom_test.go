package custom

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

func open(t *testing.T, path string) *Provider {
	t.Helper()
	p, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	p.check = func(_ context.Context, d Device) bool { return d.Address != "192.168.1.99" }
	return p
}

func TestAddDiscoverRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	p := open(t, path)
	nas, err := p.Add(Device{Name: " nas ", Kind: domain.DeviceNAS, Address: "192.168.1.20", Port: 5000})
	if err != nil || nas.Name != "nas" || nas.ID == "" {
		t.Fatalf("add: %+v %v", nas, err)
	}
	printer, err := p.Add(Device{Name: "printer", Kind: domain.DevicePrinter, Address: "192.168.1.99"})
	if err != nil {
		t.Fatal(err)
	}

	d, err := open(t, path).Discover(context.Background())
	if err != nil || len(d.Devices) != 2 || len(d.Hints) != 2 {
		t.Fatalf("devices were not saved: %+v %v", d, err)
	}
	if d.Devices[0].Status != domain.StatusUp || d.Devices[1].Status != domain.StatusDown || d.Devices[0].Addresses[0] != "192.168.1.20" {
		t.Fatalf("discovery: %+v", d.Devices)
	}
	status, _ := p.Status(context.Background())
	if status[nas.ID] != domain.StatusUp || status[printer.ID] != domain.StatusDown {
		t.Fatalf("status: %v", status)
	}
	if details, err := p.Details(context.Background(), nas.ID); err != nil || details[1].Value != "port 5000 is open" {
		t.Fatalf("details: %v %v", details, err)
	}

	if err := p.Remove(nas.ID); err != nil {
		t.Fatal(err)
	}
	if err := p.Remove(nas.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second remove: %v", err)
	}
	if d, _ := open(t, path).Discover(context.Background()); len(d.Devices) != 1 {
		t.Fatalf("remove was not saved: %+v", d.Devices)
	}
}

func TestAddRejectsBadInput(t *testing.T) {
	p := open(t, filepath.Join(t.TempDir(), "devices.json"))
	if _, err := p.Add(Device{Name: "nas", Kind: domain.DeviceNAS, Address: "nas.lan"}); err != nil {
		t.Fatal(err)
	}
	bad := []Device{
		{Name: "", Kind: domain.DeviceNAS, Address: "10.0.0.1"},
		{Name: "NAS", Kind: domain.DeviceNAS, Address: "10.0.0.1"},
		{Name: "x", Kind: "spaceship", Address: "10.0.0.1"},
		{Name: "x", Kind: domain.DeviceVM, Address: "10.0.0.1"},
		{Name: "x", Kind: domain.DeviceNAS, Address: "http://10.0.0.1"},
		{Name: "x", Kind: domain.DeviceNAS, Address: "10.0.0.1:80"},
		{Name: "x", Kind: domain.DeviceNAS, Address: "bad host"},
		{Name: "x", Kind: domain.DeviceNAS, Address: "10.0.0.1", Port: 70000},
	}
	for _, d := range bad {
		if _, err := p.Add(d); !errors.Is(err, ErrInvalid) {
			t.Errorf("%+v: got %v", d, err)
		}
	}
	if _, err := p.Add(Device{Name: "v6", Kind: domain.DeviceRouter, Address: "fd00::1"}); err != nil {
		t.Errorf("ipv6 address refused: %v", err)
	}
}
