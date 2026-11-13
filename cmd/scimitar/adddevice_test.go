package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/elliotnorth/Scimitar/internal/providers/junos"
	"golang.org/x/crypto/ssh"
)

const testHostKey = "SHA256:2bTQRn2yv7Y9tqFk9bVg1xg2n6oHq0W0n5n1o0b2c3d"

type testAdder struct {
	deviceAdder
	ti       *testInstall
	attempts int
}

func newTestAdder(t *testing.T, input string, loginErrs ...error) *testAdder {
	t.Helper()
	ti := newTestInstall(t, input)
	cfg := config{
		Listen:          "0.0.0.0:8080",
		DataDir:         ti.paths.dataDir,
		DiscoverSeconds: 300,
		StatusSeconds:   30,
		Providers:       []providerConfig{{Type: "proxmox", Settings: json.RawMessage(`{"url":"https://pve:8006","token_id":"a","token_secret":"b"}`)}},
	}
	if err := saveConfig(ti.paths.configFile(), cfg); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(ti.paths.unit, []byte("unit"), 0o644)
	ta := &testAdder{ti: ti}
	ta.deviceAdder = deviceAdder{
		prompter:   ti.prompter,
		system:     ti.system,
		configPath: ti.paths.configFile(),
		fetchHostKey: func(_ context.Context, address string) (string, string, error) {
			if strings.HasPrefix(address, "10.0.0.") {
				return testHostKey, "ssh-ed25519", nil
			}
			return "", "", errors.New("connection refused")
		},
		connect: func(_ context.Context, cfg junos.Config) (junos.SystemInfo, error) {
			ta.attempts++
			if len(loginErrs) >= ta.attempts && loginErrs[ta.attempts-1] != nil {
				return junos.SystemInfo{}, loginErrs[ta.attempts-1]
			}
			if cfg.HostKey != testHostKey || cfg.KeyFile != ti.paths.dataDir.sshKey() {
				return junos.SystemInfo{}, errors.New("wrong settings passed to connect")
			}
			return junos.SystemInfo{Hostname: "core-sw.home.lan", Model: "ex3400-48p", Version: "23.4R2"}, nil
		},
	}
	return ta
}

func TestAddDevice(t *testing.T) {
	ta := newTestAdder(t, answers("", "10.0.0.2", "y", "", ""))
	if err := ta.add(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, ta.ti.out)
	}
	out := ta.ti.out.String()
	for _, want := range []string{"set system login user scimitar authentication ssh-rsa \"ssh-rsa ", testHostKey, "EX3400-48P running Junos 23.4R2", "Added core-sw.home.lan"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	cfg, err := loadConfig(ta.ti.paths.configFile())
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Providers) != 2 || cfg.Providers[0].Type != "proxmox" || cfg.Providers[1].Type != "junos" {
		t.Fatalf("providers: %+v", cfg.Providers)
	}
	var got junos.Config
	json.Unmarshal(cfg.Providers[1].Settings, &got)
	want := junos.Config{Name: "core-sw.home.lan", Address: "10.0.0.2:22", Username: "scimitar", HostKey: testHostKey}
	if got != want {
		t.Fatalf("junos settings: %+v, want %+v", got, want)
	}
	if strings.Contains(string(cfg.Providers[1].Settings), "key_file") {
		t.Error("key_file should be left to the default")
	}
	if strings.Join(ta.ti.commands, ";") != "systemctl restart scimitar" {
		t.Fatalf("commands: %v", ta.ti.commands)
	}
	if info, err := os.Stat(ta.ti.paths.dataDir.sshKey()); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("ssh key: %v %v", info, err)
	}
	if _, err := buildRegistry(cfg.Providers, cfg.DataDir); err != nil {
		t.Fatalf("written config does not load: %v", err)
	}
}

func TestAddDeviceRetries(t *testing.T) {
	ta := newTestAdder(t, answers("", "192.168.9.9", "10.0.0.3", "y", "admin", "y", "proxmox", "bad name", "lab-switch"), errors.New("unable to authenticate"))
	if err := ta.add(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, ta.ti.out)
	}
	out := ta.ti.out.String()
	for _, want := range []string{"connection refused. Check the address", "Could not log in: unable to authenticate", "proxmox is already used", "Use letters, numbers", "Added lab-switch"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	cfg, _ := loadConfig(ta.ti.paths.configFile())
	var got junos.Config
	json.Unmarshal(cfg.Providers[1].Settings, &got)
	if got.Name != "lab-switch" || got.Username != "admin" || got.Address != "10.0.0.3:22" {
		t.Fatalf("junos settings: %+v", got)
	}
}

func TestAddDeviceStopsWhenKeyNotTrusted(t *testing.T) {
	ta := newTestAdder(t, answers("", "10.0.0.2", "n"))
	if err := ta.add(context.Background()); !errors.Is(err, errAborted) {
		t.Fatalf("got %v", err)
	}
	cfg, _ := loadConfig(ta.ti.paths.configFile())
	if len(cfg.Providers) != 1 {
		t.Fatal("config changed after cancel")
	}
}

func TestReinstallKeepsAddedDevices(t *testing.T) {
	srv := fakeProxmox(t)
	ti := newTestInstall(t, answers("y", "y", srv.URL, "", "", "good-secret", ""))
	junosSettings := json.RawMessage(`{"name":"ex3400","address":"10.0.0.2:22","username":"scimitar","host_key":"` + testHostKey + `"}`)
	old := config{
		Listen: "0.0.0.0:8080", DataDir: ti.paths.dataDir, DiscoverSeconds: 300, StatusSeconds: 30,
		Providers: []providerConfig{{Type: "proxmox", Settings: json.RawMessage(`{"url":"https://old:8006"}`)}, {Type: "junos", Settings: junosSettings}},
	}
	if err := saveConfig(ti.paths.configFile(), old); err != nil {
		t.Fatal(err)
	}
	if err := ti.install(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, ti.out)
	}
	cfg, _ := loadConfig(ti.paths.configFile())
	var kept, want junos.Config
	json.Unmarshal(junosSettings, &want)
	if len(cfg.Providers) == 2 {
		json.Unmarshal(cfg.Providers[1].Settings, &kept)
	}
	if len(cfg.Providers) != 2 || cfg.Providers[1].Type != "junos" || kept != want {
		t.Fatalf("providers after reinstall: %+v", cfg.Providers)
	}
	var pve map[string]string
	json.Unmarshal(cfg.Providers[0].Settings, &pve)
	if pve["url"] != srv.URL {
		t.Fatalf("proxmox not replaced: %v", pve)
	}
}

func TestDeviceNameFrom(t *testing.T) {
	for in, want := range map[string]string{"ex3400": "ex3400", "core sw#1": "core-sw-1", "": "juniper", "---": "juniper"} {
		if got := deviceNameFrom(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}

func TestAddDeviceReplacesUnusedEd25519Key(t *testing.T) {
	ta := newTestAdder(t, answers("", "10.0.0.2", "y", "", ""))
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "scimitar")
	if err != nil {
		t.Fatal(err)
	}
	keyPath := ta.ti.paths.dataDir.sshKey()
	if err := os.MkdirAll(string(ta.ti.paths.dataDir), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ta.add(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, ta.ti.out)
	}
	if !strings.Contains(ta.ti.out.String(), "authentication ssh-rsa \"ssh-rsa ") {
		t.Fatalf("old key was not replaced:\n%s", ta.ti.out)
	}
}

func TestAddDeviceRefusesAddressAlreadyAdded(t *testing.T) {
	first := newTestAdder(t, answers("", "10.0.0.2", "y", "", ""))
	if err := first.add(context.Background()); err != nil {
		t.Fatal(err)
	}
	first.prompter.in = bufio.NewReader(strings.NewReader(answers("", "10.0.0.2", "10.0.0.3", "y", "", "second-sw")))
	if err := first.add(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, first.ti.out)
	}
	if !strings.Contains(first.ti.out.String(), "10.0.0.2 is already in Scimitar as core-sw.home.lan") {
		t.Fatalf("no warning about the duplicate:\n%s", first.ti.out)
	}
	cfg, _ := loadConfig(first.ti.paths.configFile())
	if len(cfg.Providers) != 3 {
		t.Fatalf("providers: %+v", cfg.Providers)
	}
}

func TestRemoveDevice(t *testing.T) {
	ta := newTestAdder(t, answers("", "10.0.0.2", "y", "", ""))
	if err := ta.add(context.Background()); err != nil {
		t.Fatal(err)
	}
	ta.prompter.in = bufio.NewReader(strings.NewReader(answers("nope", "core-sw.home.lan")))
	if err := ta.remove(); err != nil {
		t.Fatalf("%v\n%s", err, ta.ti.out)
	}
	out := ta.ti.out.String()
	for _, want := range []string{"There is no device called nope", "Removed core-sw.home.lan. Scimitar restarted", "delete system login user scimitar"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	cfg, err := loadConfig(ta.ti.paths.configFile())
	if err != nil || len(cfg.Providers) != 1 || cfg.Providers[0].Type != "proxmox" {
		t.Fatalf("config after remove: %+v %v", cfg.Providers, err)
	}

	ta.ti.out.Reset()
	if err := ta.remove(); err != nil || !strings.Contains(ta.ti.out.String(), "no Juniper devices to remove") {
		t.Fatalf("nothing left: %v\n%s", err, ta.ti.out)
	}
}

func TestChoosePassword(t *testing.T) {
	out := &bytes.Buffer{}
	p := prompter{in: bufio.NewReader(strings.NewReader(answers("short", "long enough 1", "typo", "long enough 2", "long enough 2"))), out: out}
	visible := func(p prompter, question string) (string, error) { return p.required(question) }
	got, err := choosePassword(p, visible)
	if err != nil || got != "long enough 2" {
		t.Fatalf("got %q %v", got, err)
	}
	if !strings.Contains(out.String(), "too short") || !strings.Contains(out.String(), "did not match") {
		t.Fatalf("missing hints:\n%s", out)
	}
}
