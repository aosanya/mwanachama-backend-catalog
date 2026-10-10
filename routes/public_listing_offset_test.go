package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/models"
)

func TestCAT46_OpenBug_PublicListingHasNoOffsetSoRowsPastTheCapAreUnreachable(t *testing.T) {
	_, anon, cm := municipalityWard(t)
	ctx := t.Context()
	doc := `{"_item":{"ward":"Kibera North"}}`
	total := models.MaxPublicEntryLimit + 50
	for i := 0; i < total; i++ {
		slug := fmt.Sprintf("card-%03d", i)
		if _, err := cm.UpsertEntry(ctx, catalog.Entry{Slug: slug, Name: slug, Summary: slug, Doc: doc}); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
		if _, err := cm.SetVisibility(ctx, slug, catalog.VisibilityPublic); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}

	list := func(path string) []map[string]any {
		t.Helper()
		rec := do(t, anon, http.MethodGet, path, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s answered %d %s", path, rec.Code, rec.Body.String())
		}
		var rows []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
			t.Fatalf("GET %s answered a body that is not a list: %v", path, err)
		}
		return rows
	}

	page1 := list("/public/entries")
	if len(page1) != models.MaxPublicEntryLimit {
		t.Fatalf("precondition: page1 held %d rows, want the cap of %d", len(page1), models.MaxPublicEntryLimit)
	}

	page2 := list(fmt.Sprintf("/public/entries?offset=%d", models.MaxPublicEntryLimit))
	seenOnPage1 := make(map[string]bool, len(page1))
	for _, r := range page1 {
		seenOnPage1[r["slug"].(string)] = true
	}
	for _, r := range page2 {
		slug := r["slug"].(string)
		if seenOnPage1[slug] {
			t.Errorf("CAT46 open bug: GET /public/entries?offset=%d answered %q, which page 1 already carried — offset was ignored, so a caller can never reach the %d entries past the cap (seeded %d public entries total, the cap only ever lists the first %d)",
				models.MaxPublicEntryLimit, slug, total-models.MaxPublicEntryLimit, total, models.MaxPublicEntryLimit)
			break
		}
	}
	if len(page2) == 0 {
		t.Errorf("CAT46 open bug: GET /public/entries?offset=%d answered zero rows; want the %d entries past the first page", models.MaxPublicEntryLimit, total-models.MaxPublicEntryLimit)
	}
}
