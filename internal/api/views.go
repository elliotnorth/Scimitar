package api

import (
	"time"

	"github.com/elliotnorth/Scimitar/internal/core"
	"github.com/elliotnorth/Scimitar/internal/domain"
	"github.com/elliotnorth/Scimitar/internal/providers/custom"
)

type deviceView struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Kind     domain.DeviceKind `json:"kind"`
	Provider string            `json:"provider"`
	Status   domain.Status     `json:"status"`
	Address  string            `json:"address"`
	// Custom marks a device the user added by hand and can remove.
	Custom bool `json:"custom"`
}

type serviceView struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	CatalogID string        `json:"catalog_id"`
	DeviceID  string        `json:"device_id"`
	Status    domain.Status `json:"status"`
	URL       string        `json:"url"`
	// CustomURL is true when the user typed the address themselves.
	CustomURL bool `json:"custom_url"`
}

type linkView struct {
	From string          `json:"from"`
	To   string          `json:"to"`
	Kind domain.LinkKind `json:"kind"`
}

type reviewView struct {
	Key          string   `json:"key"`
	DeviceID     string   `json:"device_id"`
	Source       string   `json:"source"`
	Value        string   `json:"value"`
	CandidateIDs []string `json:"candidate_ids"`
}

type answerView struct {
	Key       string `json:"key"`
	DeviceID  string `json:"device_id"`
	Source    string `json:"source"`
	Value     string `json:"value"`
	CatalogID string `json:"catalog_id"`
}

type snapshotView struct {
	Devices   []deviceView      `json:"devices"`
	Services  []serviceView     `json:"services"`
	Links     []linkView        `json:"links"`
	Review    []reviewView      `json:"review"`
	Answered  []answerView      `json:"answered"`
	Events    []core.Event      `json:"events"`
	Providers []string          `json:"providers"`
	Errors    map[string]string `json:"errors"`
	UpdatedAt *time.Time        `json:"updated_at"`
}

type catalogView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type detailView struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func toSnapshotView(s core.Snapshot) snapshotView {
	v := snapshotView{
		Devices:   make([]deviceView, len(s.Devices)),
		Services:  make([]serviceView, len(s.Services)),
		Links:     make([]linkView, len(s.Links)),
		Review:    make([]reviewView, len(s.Review)),
		Answered:  make([]answerView, len(s.Answered)),
		Events:    append([]core.Event{}, s.Events...),
		Providers: append([]string{}, s.Providers...),
		Errors:    s.Errors,
	}
	if v.Errors == nil {
		v.Errors = map[string]string{}
	}
	if !s.UpdatedAt.IsZero() {
		v.UpdatedAt = &s.UpdatedAt
	}
	for i, d := range s.Devices {
		v.Devices[i] = deviceView{ID: d.ID, Name: d.Name, Kind: d.Kind, Provider: d.Provider, Status: d.Status, Custom: d.Provider == custom.Name}
		if len(d.Addresses) > 0 {
			v.Devices[i].Address = d.Addresses[0]
		}
	}
	for i, svc := range s.Services {
		v.Services[i] = serviceView{ID: svc.ID, Name: svc.Name, CatalogID: svc.CatalogID, DeviceID: svc.DeviceID, Status: svc.Status, URL: svc.URL, CustomURL: svc.URLSet}
	}
	for i, l := range s.Links {
		v.Links[i] = linkView{From: l.From, To: l.To, Kind: l.Kind}
	}
	for i, r := range s.Review {
		v.Review[i] = reviewView{Key: r.Key(), DeviceID: r.Hint.DeviceID, Source: r.Hint.Source, Value: r.Hint.Value, CandidateIDs: r.CandidateIDs}
	}
	for i, a := range s.Answered {
		v.Answered[i] = answerView{Key: a.Key(), DeviceID: a.Hint.DeviceID, Source: a.Hint.Source, Value: a.Hint.Value, CatalogID: a.CatalogID}
	}
	return v
}

func toCatalogView(entries []core.CatalogEntry) []catalogView {
	out := make([]catalogView, len(entries))
	for i, e := range entries {
		out[i] = catalogView{ID: e.ID, Name: e.Name}
	}
	return out
}

func toDetailViews(details []domain.Detail) []detailView {
	out := make([]detailView, len(details))
	for i, d := range details {
		out[i] = detailView{Label: d.Label, Value: d.Value}
	}
	return out
}
