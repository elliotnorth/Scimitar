package proxmox

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const (
	testTokenID     = "scimitar@pve!scimitar"
	testTokenSecret = "00000000-0000-0000-0000-000000000000"
)

var fixtureRoutes = map[string]string{
	"/api2/json/nodes":                      "nodes.json",
	"/api2/json/cluster/resources?type=vm":  "cluster_resources.json",
	"/api2/json/nodes/pve1/qemu/101/config": "qemu_101_config.json",
	"/api2/json/nodes/pve1/qemu/102/config": "qemu_102_config.json",
	"/api2/json/nodes/pve1/lxc/200/config":  "lxc_200_config.json",
}

func newFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != tokenHeader(testTokenID, testTokenSecret) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		file, ok := fixtureRoutes[r.URL.RequestURI()]
		if !ok {
			http.NotFound(w, r)
			return
		}
		body, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			t.Errorf("read fixture: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func fingerprintOf(srv *httptest.Server) string {
	return fmt.Sprintf("%X", sha256.Sum256(srv.Certificate().Raw))
}

func newTestProvider(t *testing.T, srv *httptest.Server) *Provider {
	t.Helper()
	p, err := New(Config{
		URL:         srv.URL + "/",
		TokenID:     testTokenID,
		TokenSecret: testTokenSecret,
		Fingerprint: fingerprintOf(srv),
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}
