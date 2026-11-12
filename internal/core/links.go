package core

import (
	"strings"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

// resolveLinks turns name references into device IDs and drops links whose
// target is unknown. Connected links are undirected, so a pair seen from both
// ends is kept once.
func resolveLinks(links []domain.Link, devices []domain.Device) []domain.Link {
	byName := map[string]string{}
	for _, d := range devices {
		for _, key := range nameKeys(d.Name) {
			if _, taken := byName[key]; !taken {
				byName[key] = d.ID
			}
		}
	}
	seen := map[[2]string]bool{}
	out := make([]domain.Link, 0, len(links))
	for _, l := range links {
		if name, ok := domain.ParseNameRef(l.To); ok {
			id := ""
			for _, key := range nameKeys(name) {
				if id = byName[key]; id != "" {
					break
				}
			}
			if id == "" || id == l.From {
				continue
			}
			l.To = id
		}
		if l.Kind == domain.LinkConnected {
			pair := [2]string{l.From, l.To}
			if pair[1] < pair[0] {
				pair[0], pair[1] = pair[1], pair[0]
			}
			if seen[pair] {
				continue
			}
			seen[pair] = true
		}
		out = append(out, l)
	}
	return out
}

// nameKeys returns the lowercase name and, for a domain name, its first label,
// so an LLDP neighbor called "pve.home.lan" still matches a device named "pve".
func nameKeys(name string) []string {
	full := strings.ToLower(strings.TrimSpace(name))
	if short, _, found := strings.Cut(full, "."); found && short != "" {
		return []string{full, short}
	}
	return []string{full}
}
