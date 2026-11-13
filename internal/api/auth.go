package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"github.com/elliotnorth/Scimitar/internal/auth"
)

const sessionCookie = "scimitar_session"

type Auth interface {
	Login(password string) (token string, ok bool, err error)
	Valid(token string) bool
	Logout(token string)
	SessionTTL() time.Duration
	ChangePassword(current, next string) (token string, err error)
}

func requireSession(auth Auth, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auth.Valid(sessionToken(r)) {
			writeError(w, http.StatusUnauthorized, "login required")
			return
		}
		next(w, r)
	})
}

func sessionToken(r *http.Request) string {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

type authHandlers struct {
	auth Auth
}

func (h authHandlers) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	token, ok, err := h.auth.Login(req.Password)
	switch {
	case err != nil:
		slog.Error("login failed", "err", err)
		writeError(w, http.StatusInternalServerError, "login failed")
	case !ok:
		writeError(w, http.StatusUnauthorized, "wrong password")
	default:
		h.startSession(w, token)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h authHandlers) startSession(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(h.auth.SessionTTL().Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h authHandlers) changePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	token, err := h.auth.ChangePassword(req.Current, req.New)
	switch {
	// Not 401: that means "log in again", and this person is logged in.
	case errors.Is(err, auth.ErrWrongPassword):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, auth.ErrWeakPassword):
		writeError(w, http.StatusBadRequest, err.Error())
	case err != nil:
		slog.Error("changing password failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not save the new password")
	default:
		h.startSession(w, token)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h authHandlers) logout(w http.ResponseWriter, r *http.Request) {
	h.auth.Logout(sessionToken(r))
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
	w.WriteHeader(http.StatusNoContent)
}

// Requiring a JSON content type forces a CORS preflight, so other sites cannot post to the API from a browser.
func decodeJSON(w http.ResponseWriter, r *http.Request, into any) bool {
	if mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return false
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBody)).Decode(into); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return false
	}
	return true
}
