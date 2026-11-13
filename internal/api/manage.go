package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/elliotnorth/Scimitar/internal/alerts"
	"github.com/elliotnorth/Scimitar/internal/core"
	"github.com/elliotnorth/Scimitar/internal/domain"
	"github.com/elliotnorth/Scimitar/internal/providers/custom"
)

// reason returns what the user got wrong, without the error's internal prefix.
func reason(err, kind error) string {
	return strings.TrimPrefix(err.Error(), kind.Error()+": ")
}

func (h handlers) addDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string            `json:"name"`
		Kind    domain.DeviceKind `json:"kind"`
		Address string            `json:"address"`
		Port    int               `json:"port"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	device, err := h.devices.Add(custom.Device{Name: req.Name, Kind: req.Kind, Address: req.Address, Port: req.Port})
	switch {
	case errors.Is(err, custom.ErrInvalid):
		writeError(w, http.StatusBadRequest, reason(err, custom.ErrInvalid))
	case err != nil:
		slog.Error("adding device failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not save the device")
	default:
		h.inv.DiscoverProvider(r.Context(), custom.Name)
		writeJSON(w, http.StatusCreated, map[string]string{"id": device.ID})
	}
}

func (h handlers) removeDevice(w http.ResponseWriter, r *http.Request) {
	err := h.devices.Remove(r.URL.Query().Get("id"))
	switch {
	case errors.Is(err, custom.ErrNotFound):
		writeError(w, http.StatusNotFound, "only devices you added by hand can be removed here")
	case err != nil:
		slog.Error("removing device failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not remove the device")
	default:
		h.inv.DiscoverProvider(r.Context(), custom.Name)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h handlers) setAppURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	err := h.inv.SetAppURL(req.ID, req.URL)
	switch {
	case errors.Is(err, core.ErrUnknownService):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, core.ErrInvalidAddress):
		writeError(w, http.StatusBadRequest, reason(err, core.ErrInvalidAddress))
	case err != nil:
		slog.Error("saving app address failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not save the address")
	default:
		h.inv.CheckApps(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h handlers) getAlerts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.alerts.Settings())
}

func (h handlers) putAlerts(w http.ResponseWriter, r *http.Request) {
	var req alerts.Settings
	if !decodeJSON(w, r, &req) {
		return
	}
	err := h.alerts.Update(req)
	switch {
	case errors.Is(err, alerts.ErrInvalid):
		writeError(w, http.StatusBadRequest, reason(err, alerts.ErrInvalid))
	case err != nil:
		slog.Error("saving alert settings failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not save alert settings")
	default:
		writeJSON(w, http.StatusOK, h.alerts.Settings())
	}
}

func (h handlers) testAlerts(w http.ResponseWriter, r *http.Request) {
	results := h.alerts.Test(r.Context())
	if results == nil {
		results = []alerts.Result{}
	}
	writeJSON(w, http.StatusOK, results)
}
