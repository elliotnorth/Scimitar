package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/elliotnorth/Scimitar/internal/core"
	"github.com/elliotnorth/Scimitar/internal/domain"
)

type fakeInventory struct {
	snapshot  core.Snapshot
	decideErr error
	decided   [2]string
	undone    string
	undoErr   error
	details   []domain.Detail
	detailErr error
}

func (f *fakeInventory) Snapshot() core.Snapshot { return f.snapshot }

func (f *fakeInventory) Catalog() []core.CatalogEntry {
	return []core.CatalogEntry{{ID: "immich", Name: "Immich", Aliases: []string{"immich"}}}
}

func (f *fakeInventory) Decide(key, catalogID string) error {
	f.decided = [2]string{key, catalogID}
	return f.decideErr
}

func (f *fakeInventory) Undo(key string) error {
	f.undone = key
	return f.undoErr
}

func (f *fakeInventory) Details(context.Context, string) ([]domain.Detail, error) {
	return f.details, f.detailErr
}

type fakeAuth struct {
	loggedOut string
}

const (
	testPassword = "correct horse"
	testToken    = "session-token"
)

func (f *fakeAuth) Login(password string) (string, bool, error) {
	if password != testPassword {
		return "", false, nil
	}
	return testToken, true, nil
}

func (f *fakeAuth) Valid(token string) bool { return token == testToken }

func (f *fakeAuth) Logout(token string) { f.loggedOut = token }

func (f *fakeAuth) SessionTTL() time.Duration { return time.Hour }

func newTestHandlerWithAuth(inv *fakeInventory, auth *fakeAuth) http.Handler {
	ui := fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>Scimitar</title>")}}
	return NewHandler(inv, auth, ui, Info{Version: "v1.2.3"})
}

func newTestHandler(inv *fakeInventory) http.Handler {
	return withSession{newTestHandlerWithAuth(inv, &fakeAuth{})}
}

type withSession struct{ next http.Handler }

func (s withSession) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: testToken})
	s.next.ServeHTTP(w, r)
}

func do(t *testing.T, h http.Handler, method, target, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSnapshotJSON(t *testing.T) {
	updated := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	inv := &fakeInventory{snapshot: core.Snapshot{
		Devices:   []domain.Device{{ID: "p/guest/1", Name: "immich", Kind: domain.DeviceVM, Provider: "p", Status: domain.StatusUp}},
		Services:  []domain.Service{{ID: "p/guest/1/immich", Name: "Immich", CatalogID: "immich", DeviceID: "p/guest/1", Status: domain.StatusUnknown}},
		Links:     []domain.Link{{From: "p/guest/1", To: "p/node/a", Kind: domain.LinkRunsOn}},
		Review:    []core.ReviewItem{{Hint: domain.ServiceHint{DeviceID: "p/guest/2", Source: "name", Value: "immich-old"}, CandidateIDs: []string{"immich"}}},
		Answered:  []core.Answer{{Hint: domain.ServiceHint{DeviceID: "p/guest/3", Source: "tag", Value: "hass-test"}, CatalogID: ""}},
		Providers: []string{"p", "q"},
		Errors:    map[string]string{"q": "unreachable"},
		UpdatedAt: updated,
	}}
	rec := do(t, newTestHandler(inv), http.MethodGet, "/api/snapshot", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"devices":  []any{map[string]any{"id": "p/guest/1", "name": "immich", "kind": "vm", "provider": "p", "status": "up"}},
		"services": []any{map[string]any{"id": "p/guest/1/immich", "name": "Immich", "catalog_id": "immich", "device_id": "p/guest/1", "status": "unknown"}},
		"links":    []any{map[string]any{"from": "p/guest/1", "to": "p/node/a", "kind": "runs-on"}},
		"review": []any{map[string]any{
			"key": "p/guest/2|name|immich-old", "device_id": "p/guest/2", "source": "name", "value": "immich-old",
			"candidate_ids": []any{"immich"},
		}},
		"answered": []any{map[string]any{
			"key": "p/guest/3|tag|hass-test", "device_id": "p/guest/3", "source": "tag", "value": "hass-test", "catalog_id": "",
		}},
		"providers":  []any{"p", "q"},
		"errors":     map[string]any{"q": "unreachable"},
		"updated_at": "2026-11-12T09:00:00Z",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %v\nwant %v", got, want)
	}
}

func TestEmptySnapshotUsesEmptyLists(t *testing.T) {
	rec := do(t, newTestHandler(&fakeInventory{}), http.MethodGet, "/api/snapshot", "", "")
	body := rec.Body.String()
	for _, want := range []string{`"providers":[]`, `"devices":[]`, `"review":[]`, `"answered":[]`, `"errors":{}`, `"updated_at":null`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s in %s", want, body)
		}
	}
}

func TestInfo(t *testing.T) {
	rec := do(t, newTestHandler(&fakeInventory{}), http.MethodGet, "/api/info", "", "")
	if strings.TrimSpace(rec.Body.String()) != `{"version":"v1.2.3"}` {
		t.Fatalf("got %s", rec.Body.String())
	}
}

func TestCatalog(t *testing.T) {
	rec := do(t, newTestHandler(&fakeInventory{}), http.MethodGet, "/api/catalog", "", "")
	if strings.TrimSpace(rec.Body.String()) != `[{"id":"immich","name":"Immich"}]` {
		t.Fatalf("got %s", rec.Body.String())
	}
}

func TestReview(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		body        string
		decideErr   error
		want        int
	}{
		{"saved", "application/json", `{"key":"k","catalog_id":"immich"}`, nil, http.StatusNoContent},
		{"charset allowed", "application/json; charset=utf-8", `{"key":"k","catalog_id":""}`, nil, http.StatusNoContent},
		{"form post refused", "application/x-www-form-urlencoded", `key=k`, nil, http.StatusUnsupportedMediaType},
		{"text post refused", "text/plain", `{"key":"k"}`, nil, http.StatusUnsupportedMediaType},
		{"bad json", "application/json", `{`, nil, http.StatusBadRequest},
		{"unknown item", "application/json", `{"key":"k"}`, core.ErrUnknownReviewItem, http.StatusNotFound},
		{"unknown catalog", "application/json", `{"key":"k","catalog_id":"x"}`, core.ErrUnknownCatalogEntry, http.StatusBadRequest},
		{"save failed", "application/json", `{"key":"k"}`, errors.New("disk full"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		inv := &fakeInventory{decideErr: tc.decideErr}
		rec := do(t, newTestHandler(inv), http.MethodPost, "/api/review", tc.contentType, tc.body)
		if rec.Code != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, rec.Code, tc.want)
		}
	}
}

func TestReviewPassesDecision(t *testing.T) {
	inv := &fakeInventory{}
	do(t, newTestHandler(inv), http.MethodPost, "/api/review", "application/json", `{"key":"p/guest/2|name|x","catalog_id":"immich"}`)
	if inv.decided != [2]string{"p/guest/2|name|x", "immich"} {
		t.Fatalf("got %v", inv.decided)
	}
}

func TestUndoReview(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		undoErr     error
		want        int
	}{
		{"undone", "application/json", nil, http.StatusNoContent},
		{"form post refused", "application/x-www-form-urlencoded", nil, http.StatusUnsupportedMediaType},
		{"unknown item", "application/json", core.ErrUnknownReviewItem, http.StatusNotFound},
		{"save failed", "application/json", errors.New("disk full"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		inv := &fakeInventory{undoErr: tc.undoErr}
		rec := do(t, newTestHandler(inv), http.MethodPost, "/api/review/undo", tc.contentType, `{"key":"p/guest/2|name|x"}`)
		if rec.Code != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, rec.Code, tc.want)
		}
		if tc.want == http.StatusNoContent && inv.undone != "p/guest/2|name|x" {
			t.Errorf("%s: key not passed: %q", tc.name, inv.undone)
		}
	}
}

func TestDetails(t *testing.T) {
	ok := &fakeInventory{details: []domain.Detail{{Label: "CPUs", Value: "4"}}}
	rec := do(t, newTestHandler(ok), http.MethodGet, "/api/details?id=p/guest/1", "", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `[{"label":"CPUs","value":"4"}]` {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, newTestHandler(&fakeInventory{detailErr: core.ErrUnknownDevice}), http.MethodGet, "/api/details?id=x", "", ""); rec.Code != http.StatusNotFound {
		t.Errorf("unknown device: got %d", rec.Code)
	}
	failing := &fakeInventory{detailErr: errors.New("dial tcp 10.0.0.5:8006: timeout")}
	rec = do(t, newTestHandler(failing), http.MethodGet, "/api/details?id=x", "", "")
	if rec.Code != http.StatusBadGateway || strings.Contains(rec.Body.String(), "10.0.0.5") {
		t.Errorf("provider failure: got %d %s", rec.Code, rec.Body.String())
	}
}

func TestServesUIWithSecurityHeaders(t *testing.T) {
	rec := do(t, newTestHandler(&fakeInventory{}), http.MethodGet, "/", "", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Scimitar") {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	for header, want := range map[string]string{
		"Content-Security-Policy": "default-src 'self'; frame-ancestors 'none'",
		"X-Content-Type-Options":  "nosniff",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s: got %q", header, got)
		}
	}
}

func TestWrongMethodRejected(t *testing.T) {
	if rec := do(t, newTestHandler(&fakeInventory{}), http.MethodPost, "/api/snapshot", "application/json", "{}"); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d", rec.Code)
	}
}
