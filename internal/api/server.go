package api

import (
	"context"
	"io/fs"
	"net/http"

	"github.com/elliotnorth/Scimitar/internal/alerts"
	"github.com/elliotnorth/Scimitar/internal/core"
	"github.com/elliotnorth/Scimitar/internal/domain"
	"github.com/elliotnorth/Scimitar/internal/providers/custom"
)

type Inventory interface {
	Snapshot() core.Snapshot
	Catalog() []core.CatalogEntry
	Decide(reviewKey, catalogID string) error
	Undo(reviewKey string) error
	Details(ctx context.Context, deviceID string) ([]domain.Detail, error)
	SetAppURL(serviceID, address string) error
	CheckApps(ctx context.Context)
	DiscoverProvider(ctx context.Context, name string)
}

// Devices are the devices the user adds by hand.
type Devices interface {
	Add(custom.Device) (custom.Device, error)
	Remove(id string) error
}

type Alerts interface {
	Settings() alerts.Settings
	Update(alerts.Settings) error
	Test(ctx context.Context) []alerts.Result
}

// Options are the parts of the API that can be left out.
type Options struct {
	Devices Devices
	Alerts  Alerts
}

type Info struct {
	Version     string              `json:"version"`
	DeviceKinds []domain.DeviceKind `json:"device_kinds"`
}

func NewHandler(inv Inventory, auth Auth, ui fs.FS, info Info, opts Options) http.Handler {
	info.DeviceKinds = domain.ManualKinds
	h := handlers{inv: inv, info: info, devices: opts.Devices, alerts: opts.Alerts}
	a := authHandlers{auth: auth}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", a.login)
	mux.HandleFunc("POST /api/logout", a.logout)
	mux.Handle("GET /api/snapshot", requireSession(auth, h.snapshot))
	mux.Handle("GET /api/catalog", requireSession(auth, h.catalog))
	mux.Handle("GET /api/info", requireSession(auth, h.infoHandler))
	mux.Handle("GET /api/details", requireSession(auth, h.details))
	mux.Handle("POST /api/review", requireSession(auth, h.review))
	mux.Handle("POST /api/review/undo", requireSession(auth, h.undoReview))
	mux.Handle("POST /api/password", requireSession(auth, a.changePassword))
	mux.Handle("POST /api/apps", requireSession(auth, h.setAppURL))
	if opts.Devices != nil {
		mux.Handle("POST /api/devices", requireSession(auth, h.addDevice))
		mux.Handle("DELETE /api/devices", requireSession(auth, h.removeDevice))
	}
	if opts.Alerts != nil {
		mux.Handle("GET /api/alerts", requireSession(auth, h.getAlerts))
		mux.Handle("PUT /api/alerts", requireSession(auth, h.putAlerts))
		mux.Handle("POST /api/alerts/test", requireSession(auth, h.testAlerts))
	}
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
