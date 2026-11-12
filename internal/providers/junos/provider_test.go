package junos

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

func newTestProvider(t *testing.T, dev *fakeDevice) *Provider {
	t.Helper()
	p, err := New(dev.config("ex3400"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

func TestDiscoverSwitch(t *testing.T) {
	dev := newFakeDevice(t, exReplies)
	p := newTestProvider(t, dev)
	d, err := p.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wantDevices := []domain.Device{{ID: "ex3400/device", Name: "ex3400", Kind: domain.DeviceSwitch, Provider: "ex3400", Addresses: []string{"127.0.0.1"}, Status: domain.StatusUp}}
	if !reflect.DeepEqual(d.Devices, wantDevices) {
		t.Fatalf("devices:\n got %+v\nwant %+v", d.Devices, wantDevices)
	}
	wantLinks := []domain.Link{
		{From: "ex3400/device", To: domain.NameRef("pve.home.lan"), Kind: domain.LinkConnected},
		{From: "ex3400/device", To: domain.NameRef("srx"), Kind: domain.LinkConnected},
	}
	if !reflect.DeepEqual(d.Links, wantLinks) {
		t.Fatalf("links:\n got %+v\nwant %+v", d.Links, wantLinks)
	}
}

func TestCommandsReuseOneConnection(t *testing.T) {
	dev := newFakeDevice(t, exReplies)
	p := newTestProvider(t, dev)
	for range 3 {
		if _, err := p.Status(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if n := dev.loginCount(); n != 1 {
		t.Fatalf("logged in %d times, want 1", n)
	}
}

func TestDetailsListsPorts(t *testing.T) {
	dev := newFakeDevice(t, exReplies)
	p := newTestProvider(t, dev)
	got, err := p.Details(context.Background(), "ex3400/device")
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Detail{
		{Label: "Hostname", Value: "ex3400"},
		{Label: "Model", Value: "EX3400-48P"},
		{Label: "Junos", Value: "23.4R2-S2.1"},
		{Label: "Address", Value: "127.0.0.1"},
		{Label: "Ports up", Value: "2 of 4"},
		{Label: "ge-0/0/0", Value: "up · pve mgmt · → pve.home.lan (eno1)"},
		{Label: "ge-0/0/1", Value: "off · guest desk"},
		{Label: "ge-0/0/2", Value: "empty · → srx (1)"},
		{Label: "xe-0/2/0", Value: "up · → srx (xe-0/0/1)"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("details:\n got %+v\nwant %+v", got, want)
	}
	if _, err := p.Details(context.Background(), "other/device"); err == nil {
		t.Fatal("expected error for another device")
	}
}

func TestOnlyFixedCommandsRun(t *testing.T) {
	dev := newFakeDevice(t, exReplies)
	p := newTestProvider(t, dev)
	p.Discover(context.Background())
	p.Status(context.Background())
	p.Details(context.Background(), "ex3400/device")
	allowed := map[string]bool{cmdVersion: true, cmdPorts: true, cmdDescriptions: true, cmdNeighbors: true}
	for _, c := range dev.commands {
		if !allowed[c] || !strings.HasPrefix(c, "show ") {
			t.Fatalf("ran unexpected command %q", c)
		}
	}
}

func TestNoLLDPMeansNoLinks(t *testing.T) {
	dev := newFakeDevice(t, map[string]string{cmdVersion: "ex_version.json"})
	p := newTestProvider(t, dev)
	d, err := p.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Links) != 0 || len(d.Devices) != 1 {
		t.Fatalf("unexpected discovery %+v", d)
	}
}

func TestRejectsChangedHostKey(t *testing.T) {
	dev := newFakeDevice(t, exReplies)
	cfg := dev.config("ex3400")
	cfg.HostKey = "SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	_, err = p.Discover(context.Background())
	if err == nil || !strings.Contains(err.Error(), "host key changed") {
		t.Fatalf("expected host key error, got %v", err)
	}
}

func TestRejectsWrongUser(t *testing.T) {
	dev := newFakeDevice(t, exReplies)
	cfg := dev.config("ex3400")
	cfg.Username = "admin"
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if _, err := p.Discover(context.Background()); err == nil {
		t.Fatal("expected login to fail")
	}
}

func TestFetchHostKey(t *testing.T) {
	dev := newFakeDevice(t, exReplies)
	fp, keyType, err := FetchHostKey(context.Background(), dev.addr)
	if err != nil {
		t.Fatal(err)
	}
	if fp != dev.hostKey || keyType != "ssh-ed25519" {
		t.Fatalf("got %s %s, want %s ssh-ed25519", fp, keyType, dev.hostKey)
	}
	if dev.loginCount() != 0 {
		t.Fatal("fetching the host key should not log in")
	}
}

func TestNewValidatesConfig(t *testing.T) {
	dev := newFakeDevice(t, exReplies)
	good := dev.config("ex3400")
	for name, change := range map[string]func(*Config){
		"bad name":     func(c *Config) { c.Name = "ex 3400" },
		"no address":   func(c *Config) { c.Address = "" },
		"no user":      func(c *Config) { c.Username = "" },
		"bad host key": func(c *Config) { c.HostKey = "AB:CD" },
		"no key file":  func(c *Config) { c.KeyFile = "" },
		"missing key":  func(c *Config) { c.KeyFile = filepath.Join(t.TempDir(), "nope") },
	} {
		cfg := good
		change(&cfg)
		if _, err := New(cfg); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestParseVersionShapes(t *testing.T) {
	for file, want := range map[string]SystemInfo{
		"ex_version.json":          {Hostname: "ex3400", Model: "ex3400-48p", Version: "23.4R2-S2.1"},
		"srx_cluster_version.json": {Hostname: "srx-a", Model: "srx2300", Version: "23.4R1"},
		"old_version.json":         {Hostname: "old-ex", Model: "ex2200-c-12p-2g", Version: "12.3R12.4"},
	} {
		body, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			t.Fatal(err)
		}
		got, err := parseVersion(body)
		if err != nil || got != want {
			t.Errorf("%s: got %+v, %v; want %+v", file, got, err, want)
		}
	}
	if _, err := parseVersion([]byte("error: syntax error")); err == nil {
		t.Error("expected error for non-JSON output")
	}
}

func TestKindFor(t *testing.T) {
	for model, want := range map[string]domain.DeviceKind{
		"ex3400-48p": domain.DeviceSwitch, "EX4300-48T": domain.DeviceSwitch, "qfx5120": domain.DeviceSwitch,
		"srx2300": domain.DeviceFirewall, "srx300": domain.DeviceFirewall,
		"mx204": domain.DeviceRouter, "vjunos": domain.DeviceAppliance,
	} {
		if got := kindFor(model); got != want {
			t.Errorf("%s: got %s, want %s", model, got, want)
		}
	}
}

func TestNormalizeAddress(t *testing.T) {
	for in, want := range map[string]string{
		"10.0.0.2":       "10.0.0.2:22",
		"10.0.0.2:2222":  "10.0.0.2:2222",
		"ex.home.lan":    "ex.home.lan:22",
		"fd00::2":        "[fd00::2]:22",
		"[fd00::2]:2222": "[fd00::2]:2222",
	} {
		if got, err := NormalizeAddress(in); err != nil || got != want {
			t.Errorf("%s: got %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "  ", "a b", ":22"} {
		if _, err := NormalizeAddress(bad); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}
