package api

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/elliotnorth/Scimitar/internal/core"
	"github.com/elliotnorth/Scimitar/internal/providers/custom"
)

const jsonType = "application/json"

func TestChangePassword(t *testing.T) {
	cases := []struct {
		name, body string
		want       int
	}{
		{"changed", `{"current":"right","new":"long enough"}`, http.StatusNoContent},
		{"wrong current", `{"current":"nope","new":"long enough"}`, http.StatusForbidden},
		{"too short", `{"current":"right","new":"short"}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		rec := do(t, newTestHandler(&fakeInventory{}), http.MethodPost, "/api/password", jsonType, tc.body)
		if rec.Code != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, rec.Code, tc.want)
		}
		cookie := rec.Header().Get("Set-Cookie")
		if changed := tc.want == http.StatusNoContent; changed != strings.Contains(cookie, sessionCookie+"=fresh-token") {
			t.Errorf("%s: cookie %q", tc.name, cookie)
		}
	}
	rec := do(t, newTestHandlerWithAuth(&fakeInventory{}, &fakeAuth{}), http.MethodPost, "/api/password", jsonType, `{"current":"right","new":"long enough"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("without a session: got %d", rec.Code)
	}
}

func TestAddAndRemoveDevice(t *testing.T) {
	inv := &fakeInventory{}
	h := newTestHandler(inv)
	rec := do(t, h, http.MethodPost, "/api/devices", jsonType, `{"name":"nas","kind":"nas","address":"192.168.1.20","port":5000}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"id":"custom/abc"`) {
		t.Fatalf("add: %d %s", rec.Code, rec.Body)
	}
	if testDevices.added.Port != 5000 || testDevices.added.Address != "192.168.1.20" || inv.refreshed != custom.Name {
		t.Fatalf("added %+v, refreshed %q", testDevices.added, inv.refreshed)
	}
	rec = do(t, h, http.MethodPost, "/api/devices", jsonType, `{"name":""}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"error":"give it a name"`) {
		t.Fatalf("bad device: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, http.MethodDelete, "/api/devices?id=proxmox/guest/1", "", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("removing a discovered device: %d", rec.Code)
	}
	if rec := do(t, h, http.MethodDelete, "/api/devices?id=custom/abc", "", ""); rec.Code != http.StatusNoContent || testDevices.removed != "custom/abc" {
		t.Fatalf("remove: %d", rec.Code)
	}
}

func TestSetAppURL(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"saved", nil, http.StatusNoContent},
		{"unknown app", core.ErrUnknownService, http.StatusNotFound},
		{"bad address", core.ErrInvalidAddress, http.StatusBadRequest},
		{"save failed", errors.New("disk full"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		inv := &fakeInventory{appURLErr: tc.err}
		rec := do(t, newTestHandler(inv), http.MethodPost, "/api/apps", jsonType, `{"id":"p/guest/1/immich","url":"photos.lan"}`)
		if rec.Code != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, rec.Code, tc.want)
		}
		if inv.appURL != [2]string{"p/guest/1/immich", "photos.lan"} {
			t.Errorf("%s: passed %v", tc.name, inv.appURL)
		}
		if checked := tc.err == nil; checked != (inv.checked == 1) {
			t.Errorf("%s: apps checked %d times", tc.name, inv.checked)
		}
	}
}

func TestAlerts(t *testing.T) {
	h := newTestHandler(&fakeInventory{})
	rec := do(t, h, http.MethodPut, "/api/alerts", jsonType, `{"channels":[{"type":"ntfy","url":"https://ntfy.sh/x"}],"devices":true}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"url":"https://ntfy.sh/x"`) {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, http.MethodGet, "/api/alerts", "", ""); !strings.Contains(rec.Body.String(), `"devices":true`) {
		t.Fatalf("read back: %s", rec.Body)
	}
	if rec := do(t, h, http.MethodPut, "/api/alerts", jsonType, `{"channels":[{"type":"pigeon","url":"x"}]}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad settings: %d", rec.Code)
	}
	if rec := do(t, h, http.MethodPost, "/api/alerts/test", "", ""); !strings.Contains(rec.Body.String(), `"error":"the service answered 403 Forbidden"`) {
		t.Fatalf("test: %s", rec.Body)
	}
}
