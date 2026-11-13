package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/elliotnorth/Scimitar/internal/core"
)

const maxRequestBody = 64 << 10

type handlers struct {
	inv  Inventory
	info Info
}

func (h handlers) infoHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.info)
}

func (h handlers) snapshot(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, toSnapshotView(h.inv.Snapshot()))
}

func (h handlers) catalog(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, toCatalogView(h.inv.Catalog()))
}

func (h handlers) details(w http.ResponseWriter, r *http.Request) {
	details, err := h.inv.Details(r.Context(), r.URL.Query().Get("id"))
	switch {
	case errors.Is(err, core.ErrUnknownDevice):
		writeError(w, http.StatusNotFound, err.Error())
	case err != nil:
		slog.Warn("device details failed", "err", err)
		writeError(w, http.StatusBadGateway, "provider could not load details")
	default:
		writeJSON(w, http.StatusOK, toDetailViews(details))
	}
}

type reviewRequest struct {
	Key       string `json:"key"`
	CatalogID string `json:"catalog_id"`
}

func (h handlers) review(w http.ResponseWriter, r *http.Request) {
	var req reviewRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	err := h.inv.Decide(req.Key, req.CatalogID)
	switch {
	case errors.Is(err, core.ErrUnknownReviewItem):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, core.ErrUnknownCatalogEntry):
		writeError(w, http.StatusBadRequest, err.Error())
	case err != nil:
		slog.Error("saving review decision failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not save decision")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h handlers) undoReview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key string `json:"key"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	err := h.inv.Undo(req.Key)
	switch {
	case errors.Is(err, core.ErrUnknownReviewItem):
		writeError(w, http.StatusNotFound, err.Error())
	case err != nil:
		slog.Error("undoing review decision failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not save decision")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Warn("writing response failed", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
