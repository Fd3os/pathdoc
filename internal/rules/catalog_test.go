package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every rule must say what the user sees and link its evidence.
func TestCatalogComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range Catalog {
		if seen[r.ID] {
			t.Errorf("duplicate rule %s", r.ID)
		}
		seen[r.ID] = true
		if r.Title == "" || r.Symptom == "" || r.Breaks == "" || len(r.Locations) == 0 || r.Applies == nil {
			t.Errorf("%s: missing title, symptom, breaks, locations or applies", r.ID)
		}
		if len(r.Evidence) == 0 {
			t.Errorf("%s: no evidence", r.ID)
		}
		if r.Fix == FixEnv && len(r.Env) == 0 {
			t.Errorf("%s: env fix with no variables", r.ID)
		}
	}
	if len(Catalog) != 6 {
		t.Errorf("catalog has %d rules, want the 6 the lab reproduced", len(Catalog))
	}
}

// A rule without a repro test does not ship.
func TestCatalogHasRepro(t *testing.T) {
	t.Skip("repro tests arrive in Task 7")
	for _, r := range Catalog {
		b, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(r.Repro)))
		if err != nil {
			t.Errorf("%s: repro file %q: %v", r.ID, r.Repro, err)
			continue
		}
		fn := "func TestRepro_" + strings.ReplaceAll(r.ID, "-", "_") + "("
		if !strings.Contains(string(b), fn) {
			t.Errorf("%s: %s has no %s", r.ID, r.Repro, fn)
		}
	}
}
