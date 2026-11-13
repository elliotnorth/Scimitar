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

var (
	ErrUnknownReviewItem   = errors.New("no such review item")
	ErrUnknownCatalogEntry = errors.New("no such catalog entry")
)

// applyDecisions turns decided review items into services and returns them as
// answers, so the user can see and undo what they chose.
func (r *MatchResult) applyDecisions(catalog []CatalogEntry, decisions Decisions) []Answer {
	var answers []Answer
	seen := map[string]bool{}
	undecided := r.Review[:0]
	for _, item := range r.Review {
		catalogID, decided := decisions[item.Key()]
		if !decided {
			undecided = append(undecided, item)
			continue
		}
		if !seen[item.Key()] {
			seen[item.Key()] = true
			answers = append(answers, Answer{Hint: item.Hint, CatalogID: catalogID})
		}
		if entry, ok := findEntry(catalog, catalogID); ok {
			r.addService(item.Hint.DeviceID, entry)
		}
	}
	r.Review = undecided
	r.pruneConfirmed()
	return answers
}

func validateDecision(catalog []CatalogEntry, catalogID string) error {
	if catalogID == NotAService {
		return nil
	}
	if _, ok := findEntry(catalog, catalogID); !ok {
		return fmt.Errorf("%w: %q", ErrUnknownCatalogEntry, catalogID)
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
