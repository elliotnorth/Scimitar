package api

import (
	"encoding/json"
	"log/slog"
	"mime"
	"net/http"
	"time"
)

const sessionCookie = "scimitar_session"

type Auth interface {
	Login(password string) (token string, ok bool, err error)
	Valid(token string) bool
	Logout(token string)
	SessionTTL() time.Duration
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
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookie,
			Value:    token,
			Path:     "/",
			MaxAge:   int(h.auth.SessionTTL().Seconds()),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		})
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
