package probe

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckHTTP(t *testing.T) {
	status := http.StatusOK
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) })
	plain := httptest.NewServer(handler)
	defer plain.Close()
	secure := httptest.NewTLSServer(handler)
	defer secure.Close()
	p := New()
	ctx := context.Background()

	for _, code := range []int{200, 302, 401, 404} {
		status = code
		if !p.Check(ctx, plain.URL) {
			t.Errorf("status %d should count as up", code)
		}
	}
	status = http.StatusBadGateway
	if p.Check(ctx, plain.URL) {
		t.Error("502 should count as down")
	}
	status = http.StatusOK
	if !p.Check(ctx, secure.URL) {
		t.Error("self-signed https should count as up")
	}
	if !p.Check(ctx, "tcp://"+strings.TrimPrefix(plain.URL, "http://")) {
		t.Error("open tcp port should count as up")
	}
	plain.Close()
	if p.Check(ctx, plain.URL) {
		t.Error("closed server should count as down")
	}
}

func TestReachable(t *testing.T) {
	// Nothing listens on these ports on loopback, but the host refuses, which proves it is up.
	if !Reachable(context.Background(), "127.0.0.1") {
		t.Error("loopback should be reachable")
	}
	// 192.0.2.0/24 is reserved for documentation and never answers.
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout/6)
	defer cancel()
	if Reachable(ctx, "192.0.2.1") {
		t.Error("unroutable address should not be reachable")
	}
}

func TestPortOpen(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if !PortOpen(context.Background(), "127.0.0.1", port) {
		t.Error("listening port should be open")
	}
	l.Close()
	if PortOpen(context.Background(), "127.0.0.1", port) {
		t.Error("closed port should not be open")
	}
}

func TestNormalizeTarget(t *testing.T) {
	good := map[string]string{
		"192.168.1.5:8096":       "http://192.168.1.5:8096",
		" https://nas.lan/ ":     "https://nas.lan/",
		"tcp://192.168.1.5:1883": "tcp://192.168.1.5:1883",
	}
	for in, want := range good {
		if got, err := NormalizeTarget(in); err != nil || got != want {
			t.Errorf("%q: got %q %v, want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "ftp://x", "tcp://host", "http://", "http://user:pw@host", "file:///etc/passwd"} {
		if _, err := NormalizeTarget(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
