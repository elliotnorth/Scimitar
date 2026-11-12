package core

import (
	"fmt"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

type Registry struct {
	providers []domain.Provider
}

func (r *Registry) Register(p domain.Provider) error {
	for _, existing := range r.providers {
		if existing.Name() == p.Name() {
			return fmt.Errorf("provider %q already registered", p.Name())
		}
	}
	r.providers = append(r.providers, p)
	return nil
}

func (r *Registry) Providers() []domain.Provider {
	return append([]domain.Provider(nil), r.providers...)
}
