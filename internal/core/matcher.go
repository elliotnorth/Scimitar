package core

import (
	"strings"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

type MatchResult struct {
	Services []domain.Service
	Review   []ReviewItem
}

func Match(catalog []CatalogEntry, hints []domain.ServiceHint) MatchResult {
	var result MatchResult
	for _, hint := range hints {
		exact, partial := matchHint(catalog, hint.Value)
		switch {
		case len(exact) == 1:
			result.addService(hint.DeviceID, exact[0])
		case len(exact) > 1 || len(partial) > 0:
			result.Review = append(result.Review, ReviewItem{Hint: hint, CandidateIDs: entryIDs(append(exact, partial...))})
		}
	}
	result.pruneConfirmed()
	return result
}

func (r *MatchResult) addService(deviceID string, entry CatalogEntry) {
	id := serviceID(deviceID, entry.ID)
	for _, s := range r.Services {
		if s.ID == id {
			return
		}
	}
	r.Services = append(r.Services, newService(deviceID, entry))
}

func matchHint(catalog []CatalogEntry, value string) (exact, partial []CatalogEntry) {
	v := normalize(value)
	if v == "" {
		return nil, nil
	}
	for _, entry := range catalog {
		switch aliasMatch(entry, v) {
		case matchExact:
			exact = append(exact, entry)
		case matchPartial:
			partial = append(partial, entry)
		}
	}
	return exact, partial
}

type matchKind int

const (
	matchNone matchKind = iota
	matchPartial
	matchExact
)

func aliasMatch(entry CatalogEntry, normalizedValue string) matchKind {
	best := matchNone
	for _, alias := range entry.Aliases {
		a := normalize(alias)
		if a == normalizedValue {
			return matchExact
		}
		if strings.Contains(normalizedValue, a) {
			best = matchPartial
		}
	}
	return best
}

func newService(deviceID string, entry CatalogEntry) domain.Service {
	return domain.Service{
		ID:        serviceID(deviceID, entry.ID),
		Name:      entry.Name,
		CatalogID: entry.ID,
		DeviceID:  deviceID,
		Status:    domain.StatusUnknown,
	}
}

func serviceID(deviceID, catalogID string) string {
	return deviceID + "/" + catalogID
}

func entryIDs(entries []CatalogEntry) []string {
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.ID
	}
	return ids
}
