package alerts

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/elliotnorth/Scimitar/internal/core"
	"github.com/elliotnorth/Scimitar/internal/domain"
)

type received struct {
	path, title, priority, body string
}

func catcher(t *testing.T) (*httptest.Server, func() []received) {
	t.Helper()
	var mu sync.Mutex
	var got []received
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = append(got, received{r.URL.Path, r.Header.Get("Title"), r.Header.Get("Priority"), string(body)})
		mu.Unlock()
		if r.URL.Path == "/broken" {
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, func() []received {
		mu.Lock()
		defer mu.Unlock()
		return append([]received(nil), got...)
	}
}

func TestSettingsAreCheckedAndSaved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "alerts.json")
	n, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if s := n.Settings(); len(s.Channels) != 0 || !s.Devices || !s.Apps || !s.Sources {
		t.Fatalf("defaults: %+v", s)
	}
	for _, bad := range []Settings{
		{Channels: []Channel{{Type: "pigeon", URL: "https://x"}}},
		{Channels: []Channel{{Type: TypeNtfy, URL: "ntfy.sh/topic"}}},
		{Channels: []Channel{{Type: TypeNtfy, URL: "file:///etc/passwd"}}},
		{Channels: make([]Channel, maxChannels+1)},
	} {
		if err := n.Update(bad); !errors.Is(err, ErrInvalid) {
			t.Errorf("%+v: got %v", bad, err)
		}
	}
	want := Settings{Channels: []Channel{{Type: TypeNtfy, URL: " https://ntfy.sh/my-topic "}}, Devices: true}
	if err := n.Update(want); err != nil {
		t.Fatal(err)
	}
	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if s := again.Settings(); len(s.Channels) != 1 || s.Channels[0].URL != "https://ntfy.sh/my-topic" || !s.Devices || s.Apps {
		t.Fatalf("not saved: %+v", s)
	}
}

func TestEachTypeSendsItsOwnFormat(t *testing.T) {
	srv, got := catcher(t)
	n, _ := Open(filepath.Join(t.TempDir(), "alerts.json"))
	channels := []Channel{}
	for _, kind := range []string{TypeNtfy, TypeDiscord, TypeSlack, TypeGotify, TypeWebhook} {
		channels = append(channels, Channel{Type: kind, URL: srv.URL + "/" + kind})
	}
	if err := n.Update(Settings{Channels: channels, Devices: true}); err != nil {
		t.Fatal(err)
	}
	down := core.Event{Time: time.Unix(0, 0), Subject: core.SubjectDevice, ID: "p/guest/1", Name: "nas", Status: domain.StatusDown, Detail: "nas on custom"}
	app := core.Event{Subject: core.SubjectApp, Name: "Immich", Status: domain.StatusDown}
	n.Notify([]core.Event{down, app})

	deadline := time.Now().Add(3 * time.Second)
	for len(got()) < len(channels) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	byPath := map[string]received{}
	for _, r := range got() {
		byPath[r.path] = r
	}
	if len(got()) != len(channels) {
		t.Fatalf("got %d messages, want %d: app events are switched off", len(got()), len(channels))
	}
	if r := byPath["/ntfy"]; r.title != "nas is down" || r.priority != "high" || r.body != "nas on custom" {
		t.Errorf("ntfy: %+v", r)
	}
	for path, field := range map[string]string{"/discord": "content", "/slack": "text", "/gotify": "message", "/webhook": "title"} {
		var body map[string]any
		if err := json.Unmarshal([]byte(byPath[path].body), &body); err != nil {
			t.Errorf("%s: not json: %q", path, byPath[path].body)
			continue
		}
		if text, _ := body[field].(string); !strings.HasPrefix(text, "nas is down") {
			t.Errorf("%s: %v", path, body)
		}
	}
	var hook map[string]any
	json.Unmarshal([]byte(byPath["/webhook"].body), &hook)
	if hook["status"] != "down" || hook["subject"] != "device" || hook["id"] != "p/guest/1" {
		t.Errorf("webhook: %v", hook)
	}
}

func TestTestReportsEachChannel(t *testing.T) {
	srv, _ := catcher(t)
	n, _ := Open(filepath.Join(t.TempDir(), "alerts.json"))
	n.Update(Settings{Channels: []Channel{
		{Type: TypeNtfy, URL: srv.URL + "/ok"},
		{Type: TypeDiscord, URL: srv.URL + "/broken"},
		{Type: TypeSlack, URL: "http://127.0.0.1:1/secret-token"},
	}})
	results := n.Test(context.Background())
	if len(results) != 3 || results[0].Error != "" || !strings.Contains(results[1].Error, "403") || results[2].Error == "" {
		t.Fatalf("results: %+v", results)
	}
	if strings.Contains(results[2].Error, "secret-token") {
		t.Fatalf("error leaks the address: %s", results[2].Error)
	}
}

func TestMessages(t *testing.T) {
	cases := map[string]core.Event{
		"nas is down":                      {Subject: core.SubjectDevice, Name: "nas", Status: domain.StatusDown},
		"Immich is back up":                {Subject: core.SubjectApp, Name: "Immich", Status: domain.StatusUp},
		"Scimitar can't reach proxmox":     {Subject: core.SubjectSource, Name: "proxmox", Status: domain.StatusDown},
		"Scimitar can reach proxmox again": {Subject: core.SubjectSource, Name: "proxmox", Status: domain.StatusUp},
	}
	for want, e := range cases {
		if got := describe(e).Title; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
