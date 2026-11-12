package api

import (
	"net/http"
	"testing"
)

func TestAPIRequiresSession(t *testing.T) {
	h := newTestHandlerWithAuth(&fakeInventory{}, &fakeAuth{})
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/snapshot"},
		{http.MethodGet, "/api/catalog"},
		{http.MethodGet, "/api/details?id=x"},
		{http.MethodPost, "/api/review"},
	} {
		if rec := do(t, h, tc.method, tc.path, "application/json", `{}`); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: got %d", tc.method, tc.path, rec.Code)
		}
	}
}

func TestUIIsPublic(t *testing.T) {
	h := newTestHandlerWithAuth(&fakeInventory{}, &fakeAuth{})
	if rec := do(t, h, http.MethodGet, "/", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestLoginSetsStrictCookie(t *testing.T) {
	h := newTestHandlerWithAuth(&fakeInventory{}, &fakeAuth{})
	rec := do(t, h, http.MethodPost, "/api/login", "application/json", `{"password":"correct horse"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("got %d", rec.Code)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies: %v", cookies)
	}
	c := cookies[0]
	if c.Name != sessionCookie || c.Value != testToken || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.MaxAge != 3600 {
		t.Fatalf("cookie: %+v", c)
	}
}

func TestLoginFailures(t *testing.T) {
	h := newTestHandlerWithAuth(&fakeInventory{}, &fakeAuth{})
	cases := []struct {
		contentType, body string
		want              int
	}{
		{"application/json", `{"password":"wrong"}`, http.StatusUnauthorized},
		{"application/x-www-form-urlencoded", `password=correct+horse`, http.StatusUnsupportedMediaType},
		{"application/json", `{`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		rec := do(t, h, http.MethodPost, "/api/login", tc.contentType, tc.body)
		if rec.Code != tc.want || len(rec.Result().Cookies()) != 0 {
			t.Errorf("%s %s: got %d with cookies %v", tc.contentType, tc.body, rec.Code, rec.Result().Cookies())
		}
	}
}

func TestLogoutEndsSession(t *testing.T) {
	auth := &fakeAuth{}
	h := withSession{newTestHandlerWithAuth(&fakeInventory{}, auth)}
	rec := do(t, h, http.MethodPost, "/api/logout", "", "")
	if rec.Code != http.StatusNoContent || auth.loggedOut != testToken {
		t.Fatalf("got %d, logged out %q", rec.Code, auth.loggedOut)
	}
	if c := rec.Result().Cookies(); len(c) != 1 || c[0].MaxAge >= 0 {
		t.Fatalf("cookie not cleared: %v", c)
	}
}
