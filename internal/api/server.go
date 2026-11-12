package api

import (
	"context"
	"io/fs"
	"net/http"

	"github.com/elliotnorth/Scimitar/internal/core"
	"github.com/elliotnorth/Scimitar/internal/domain"
)

type Inventory interface {
	Snapshot() core.Snapshot
	Catalog() []core.CatalogEntry
	Decide(reviewKey, catalogID string) error
	Details(ctx context.Context, deviceID string) ([]domain.Detail, error)
}

func NewHandler(inv Inventory, auth Auth, ui fs.FS) http.Handler {
	h := handlers{inv: inv}
	a := authHandlers{auth: auth}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", a.login)
	mux.HandleFunc("POST /api/logout", a.logout)
	mux.Handle("GET /api/snapshot", requireSession(auth, h.snapshot))
	mux.Handle("GET /api/catalog", requireSession(auth, h.catalog))
	mux.Handle("GET /api/details", requireSession(auth, h.details))
	mux.Handle("POST /api/review", requireSession(auth, h.review))
	mux.Handle("GET /", http.FileServerFS(ui))
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
