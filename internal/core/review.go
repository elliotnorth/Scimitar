package core

import "github.com/elliotnorth/Scimitar/internal/domain"

type ReviewItem struct {
	Hint         domain.ServiceHint
	CandidateIDs []string
}

func (r ReviewItem) Key() string {
	return r.Hint.DeviceID + "|" + r.Hint.Source + "|" + r.Hint.Value
}

// Answer is a review item the user has already decided. CatalogID is
// NotAService when they said it is not an app.
type Answer struct {
	Hint      domain.ServiceHint
	CatalogID string
}

func (a Answer) Key() string {
	return ReviewItem{Hint: a.Hint}.Key()
}

func (r *MatchResult) pruneConfirmed() {
	confirmed := map[string]bool{}
	for _, s := range r.Services {
		confirmed[s.ID] = true
	}
	open := r.Review[:0]
	for _, item := range r.Review {
		var candidates []string
		for _, id := range item.CandidateIDs {
			if !confirmed[serviceID(item.Hint.DeviceID, id)] {
				candidates = append(candidates, id)
			}
		}
		if len(candidates) > 0 {
			item.CandidateIDs = candidates
			open = append(open, item)
		}
	}
	r.Review = open
}
