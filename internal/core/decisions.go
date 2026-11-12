package core

import (
	"errors"
	"fmt"
)

const NotAService = ""

type Decisions map[string]string

type DecisionStore interface {
	Load() (Decisions, error)
	Save(Decisions) error
}

var ErrUnknownReviewItem = errors.New("no such review item")

func (r *MatchResult) applyDecisions(catalog []CatalogEntry, decisions Decisions) {
	undecided := r.Review[:0]
	for _, item := range r.Review {
		catalogID, decided := decisions[item.Key()]
		if !decided {
			undecided = append(undecided, item)
			continue
		}
		if entry, ok := findEntry(catalog, catalogID); ok {
			r.addService(item.Hint.DeviceID, entry)
		}
	}
	r.Review = undecided
	r.pruneConfirmed()
}

func validateDecision(catalog []CatalogEntry, catalogID string) error {
	if catalogID == NotAService {
		return nil
	}
	if _, ok := findEntry(catalog, catalogID); !ok {
		return fmt.Errorf("unknown catalog id %q", catalogID)
	}
	return nil
}

func findEntry(catalog []CatalogEntry, id string) (CatalogEntry, bool) {
	for _, entry := range catalog {
		if entry.ID == id {
			return entry, true
		}
	}
	return CatalogEntry{}, false
}
