package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT22_OpenBug_SearchTreatsUnderscoreAndPercentAsWildcards(t *testing.T) {
	h, cm, ctx := newServer(t)

	for i, name := range []string{"Alpha", "Beta", "Gamma"} {
		slug := fmt.Sprintf("entry-%d", i)
		rec := do(t, h, http.MethodPut, "/entries/"+slug,
			fmt.Sprintf(`{"name":%q,"summary":"a summary","doc":%s}`, name, mustJSON(doc)))
		if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
			t.Fatalf("seed %s: status = %d, body %s", slug, rec.Code, rec.Body.String())
		}
		if _, err := cm.SetVisibility(ctx, slug, catalog.VisibilityPublic); err != nil {
			t.Fatalf("set visibility %s: %v", slug, err)
		}
	}

	rec := do(t, h, http.MethodGet, "/entries?search=_", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: status = %d, body %s", rec.Code, rec.Body.String())
	}
	var rows []catalog.Entry
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("CAT22 open bug: GET /entries?search=_ matched %d of 3 seeded entries (Alpha, Beta, Gamma — none contains a literal underscore); want 0, because the caller's search term is a literal substring, not a SQL LIKE pattern, and '_' must not act as a single-character wildcard that matches every row", len(rows))
	}
}

func mustJSON(doc string) string {
	b, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return string(b)
}
