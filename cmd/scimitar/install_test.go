package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elliotnorth/Scimitar/deploy"
	"github.com/elliotnorth/Scimitar/internal/auth"
)

const fixtureDir = "../../internal/providers/proxmox/testdata"

func fakeProxmox(t *testing.T) *httptest.Server {
	t.Helper()
	routes := map[string]string{
		"/api2/json/nodes":                      "nodes.json",
		"/api2/json/cluster/resources?type=vm":  "cluster_resources.json",
		"/api2/json/nodes/pve1/qemu/101/config": "qemu_101_config.json",
		"/api2/json/nodes/pve1/qemu/102/config": "qemu_102_config.json",
		"/api2/json/nodes/pve1/lxc/200/config":  "lxc_200_config.json",
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "PVEAPIToken=scimitar@pve!scimitar=good-secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		body, err := os.ReadFile(filepath.Join(fixtureDir, routes[r.URL.RequestURI()]))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

type testInstall struct {
	installer
	out       *bytes.Buffer
	commands  []string
	busy      map[string]bool
	healthErr error
}

func newTestInstall(t *testing.T, input string) *testInstall {
	t.Helper()
	root := t.TempDir()
	self := filepath.Join(root, "build", "scimitar")
	os.MkdirAll(filepath.Dir(self), 0o755)
	os.WriteFile(self, []byte("binary"), 0o755)
	for _, dir := range []string{"bin", "systemd"} {
		os.MkdirAll(filepath.Join(root, dir), 0o755)
	}
	ti := &testInstall{out: &bytes.Buffer{}, busy: map[string]bool{}}
	ti.installer = installer{
		prompter: prompter{in: bufio.NewReader(strings.NewReader(input)), out: ti.out},
		system: system{
			paths: installPaths{
				binary:    filepath.Join(root, "bin", "scimitar"),
				configDir: filepath.Join(root, "etc"),
				unit:      filepath.Join(root, "systemd", "scimitar.service"),
				dataDir:   dataDir(filepath.Join(root, "data")),
			},
			run: func(name string, args ...string) error {
				ti.commands = append(ti.commands, name+" "+strings.Join(args, " "))
				return nil
			},
			owner:      func() (int, int, error) { return os.Getuid(), os.Getgid(), nil },
			executable: func() (string, error) { return self, nil },
			portFree:   func(listen string) bool { return !ti.busy[listen] },
			healthy:    func(context.Context, string) error { return ti.healthErr },
		},
		readSecret: func(p prompter, q string) (string, error) { return p.required(q) },
	}
	return ti
}

func answers(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func TestInstallWithExistingToken(t *testing.T) {
	srv := fakeProxmox(t)
	ti := newTestInstall(t, answers("y", strings.TrimPrefix(srv.URL, "https://"), "", "", "good-secret", ""))
	if err := ti.install(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, ti.out)
	}
	out := ti.out.String()
	for _, want := range []string{"Found 2 hosts and 4 VMs and containers", "Scimitar is running", "Password:  "} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	cfg, err := loadConfig(ti.paths.configFile())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "0.0.0.0:8080" || cfg.DataDir != ti.paths.dataDir {
		t.Fatalf("config: %+v", cfg)
	}
	var pve map[string]string
	json.Unmarshal(cfg.Providers[0].Settings, &pve)
	wantFP := colonHex(sha256.Sum256(srv.Certificate().Raw))
	if pve["url"] != srv.URL || pve["token_id"] != "scimitar@pve!scimitar" || pve["token_secret"] != "good-secret" || pve["fingerprint"] != wantFP {
		t.Fatalf("proxmox settings: %v", pve)
	}
	if _, err := buildRegistry(cfg.Providers, cfg.DataDir); err != nil {
		t.Fatalf("written config does not load: %v", err)
	}

	if info, _ := os.Stat(ti.paths.configFile()); info.Mode().Perm() != 0o600 {
		t.Errorf("config mode %v", info.Mode().Perm())
	}
	if unit, _ := os.ReadFile(ti.paths.unit); string(unit) != deploy.SystemdUnit {
		t.Error("unit file not written")
	}
	if bin, _ := os.ReadFile(ti.paths.binary); string(bin) != "binary" {
		t.Error("binary not copied")
	}
	password := lineValue(out, "Password:")
	hash, _, err := auth.LoadOrCreate(ti.paths.dataDir.auth())
	if err != nil || !hash.Matches(password) {
		t.Fatalf("printed password does not match stored hash: %v", err)
	}
	wantCommands := []string{"systemctl daemon-reload", "systemctl enable scimitar", "systemctl restart scimitar"}
	if strings.Join(ti.commands, ";") != strings.Join(wantCommands, ";") {
		t.Fatalf("commands: %v", ti.commands)
	}
}

func TestInstallShowsTokenStepsAndLocalOnly(t *testing.T) {
	srv := fakeProxmox(t)
	ti := newTestInstall(t, answers("n", "", srv.URL, "y", "", "good-secret", "n"))
	if err := ti.install(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, ti.out)
	}
	if !strings.Contains(ti.out.String(), "pveum user token add scimitar@pve scimitar --privsep 1") {
		t.Error("token commands not shown")
	}
	cfg, _ := loadConfig(ti.paths.configFile())
	if cfg.Listen != "127.0.0.1:8080" {
		t.Fatalf("listen: %s", cfg.Listen)
	}
}

func TestInstallRetriesWrongSecret(t *testing.T) {
	srv := fakeProxmox(t)
	ti := newTestInstall(t, answers("y", srv.URL, "", "", "bad-secret", "", "", "good-secret", ""))
	if err := ti.install(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, ti.out)
	}
	if !strings.Contains(ti.out.String(), "401") {
		t.Fatalf("failure not explained:\n%s", ti.out)
	}
}

func TestInstallRetriesBadAddress(t *testing.T) {
	srv := fakeProxmox(t)
	ti := newTestInstall(t, answers("y", "http://"+strings.TrimPrefix(srv.URL, "https://"), "127.0.0.1:1", srv.URL, "", "", "good-secret", ""))
	if err := ti.install(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, ti.out)
	}
	if got := strings.Count(ti.out.String(), "Check the address and try again"); got != 2 {
		t.Fatalf("expected two retries, got %d:\n%s", got, ti.out)
	}
}

func TestInstallPicksAnotherPortWhenBusy(t *testing.T) {
	srv := fakeProxmox(t)
	ti := newTestInstall(t, answers("y", srv.URL, "", "", "good-secret", "", "9090"))
	ti.busy["0.0.0.0:8080"] = true
	if err := ti.install(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, ti.out)
	}
	cfg, _ := loadConfig(ti.paths.configFile())
	if cfg.Listen != "0.0.0.0:9090" || !strings.Contains(ti.out.String(), ":9090") {
		t.Fatalf("listen %s\n%s", cfg.Listen, ti.out)
	}
}

func TestInstallReportsServiceThatDoesNotStart(t *testing.T) {
	srv := fakeProxmox(t)
	ti := newTestInstall(t, answers("y", srv.URL, "", "", "good-secret", ""))
	ti.healthErr = errors.New("connection refused")
	if err := ti.install(context.Background()); !errors.Is(err, errAborted) {
		t.Fatalf("got %v", err)
	}
	if out := ti.out.String(); !strings.Contains(out, "journalctl -u scimitar") || strings.Contains(out, "Scimitar is running") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestInstallStopsWhenTokenRetryDeclined(t *testing.T) {
	srv := fakeProxmox(t)
	ti := newTestInstall(t, answers("y", srv.URL, "", "", "bad-secret", "n"))
	if err := ti.install(context.Background()); !errors.Is(err, errAborted) {
		t.Fatalf("got %v", err)
	}
	if fileExists(ti.paths.configFile()) || len(ti.commands) != 0 {
		t.Fatal("installer changed the system after a failed check")
	}
}

func TestInstallStopsWhenCertificateRejected(t *testing.T) {
	srv := fakeProxmox(t)
	ti := newTestInstall(t, answers("y", srv.URL, "n"))
	if err := ti.install(context.Background()); !errors.Is(err, errAborted) {
		t.Fatalf("got %v", err)
	}
}

func TestReinstallKeepsPassword(t *testing.T) {
	srv := fakeProxmox(t)
	ti := newTestInstall(t, answers("y", srv.URL, "", "", "good-secret", ""))
	if err := ti.install(context.Background()); err != nil {
		t.Fatal(err)
	}
	first := lineValue(ti.out.String(), "Password:")

	again := newTestInstall(t, answers("y", "y", srv.URL, "", "", "good-secret", ""))
	again.system = ti.system
	if err := again.install(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(again.out.String(), "unchanged from your previous install") {
		t.Fatalf("output:\n%s", again.out)
	}
	hash, _, _ := auth.LoadOrCreate(ti.paths.dataDir.auth())
	if !hash.Matches(first) {
		t.Fatal("reinstall changed the password")
	}
}

func TestReinstallCanBeDeclined(t *testing.T) {
	ti := newTestInstall(t, answers("n"))
	os.MkdirAll(ti.paths.configDir, 0o700)
	os.WriteFile(ti.paths.configFile(), []byte("{}"), 0o600)
	if err := ti.install(context.Background()); !errors.Is(err, errAborted) {
		t.Fatalf("got %v", err)
	}
}

func TestUninstall(t *testing.T) {
	srv := fakeProxmox(t)
	ti := newTestInstall(t, answers("y", srv.URL, "", "", "good-secret", ""))
	if err := ti.install(context.Background()); err != nil {
		t.Fatal(err)
	}
	un := newTestInstall(t, answers("y"))
	un.system = ti.system
	if err := un.uninstall(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{ti.paths.unit, ti.paths.binary, ti.paths.configDir} {
		if fileExists(path) {
			t.Errorf("%s still exists", path)
		}
	}
	if !fileExists(ti.paths.dataDir.auth()) {
		t.Error("data was deleted")
	}
}

func TestProxmoxURL(t *testing.T) {
	cases := map[string]string{
		"192.168.1.10":          "https://192.168.1.10:8006",
		"pve.lan:8443":          "https://pve.lan:8443",
		"https://pve.lan":       "https://pve.lan:8006",
		"https://pve.lan:8006/": "https://pve.lan:8006",
		"fd00::10":              "",
		"[fd00::10]":            "https://[fd00::10]:8006",
		"http://192.168.1.10":   "",
		"":                      "",
	}
	for in, want := range cases {
		got, err := proxmoxURL(in)
		if want == "" {
			if err == nil {
				t.Errorf("%q: expected error, got %q", in, got)
			}
			continue
		}
		if err != nil || got != want {
			t.Errorf("%q: got %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestPromptHandlesEOF(t *testing.T) {
	p := prompter{in: bufio.NewReader(strings.NewReader("")), out: &bytes.Buffer{}}
	if _, err := p.required("anything"); !errors.Is(err, errAborted) {
		t.Fatalf("got %v", err)
	}
}

func lineValue(out, label string) string {
	for _, line := range strings.Split(out, "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), label); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}
