package core

import "testing"

func TestDefaultCatalogAliasesAreUnique(t *testing.T) {
	owner := map[string]string{}
	for _, entry := range DefaultCatalog {
		if len(entry.Aliases) == 0 {
			t.Errorf("%s has no aliases", entry.ID)
		}
		for _, alias := range entry.Aliases {
			key := normalize(alias)
			if prev, taken := owner[key]; taken {
				t.Errorf("alias %q used by %s and %s", alias, prev, entry.ID)
			}
			owner[key] = entry.ID
		}
	}
}
