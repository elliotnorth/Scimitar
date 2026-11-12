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

func NewHandler(inv Inventory, ui fs.FS) http.Handler {
	h := handlers{inv: inv}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/snapshot", h.snapshot)
	mux.HandleFunc("GET /api/catalog", h.catalog)
	mux.HandleFunc("GET /api/details", h.details)
	mux.HandleFunc("POST /api/review", h.review)
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
