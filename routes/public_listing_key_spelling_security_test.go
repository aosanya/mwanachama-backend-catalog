package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/models"
)

func TestAnonymousListingStaysPublicUnderEveryQueryKeySpelling(t *testing.T) {
	operator, anon, cm := municipalityWard(t)
	ctx := t.Context()
	doc := `{"_item":{"ward":"Kibera North"}}`
	put := func(slug string, v catalog.Visibility) {
		t.Helper()
		if _, err := cm.UpsertEntry(ctx, catalog.Entry{Slug: slug, Name: slug, Summary: slug, Doc: doc}); err != nil {
			t.Fatalf("precondition: UpsertEntry(%s): %v", slug, err)
		}
		if v != "" {
			if _, err := cm.SetVisibility(ctx, slug, v); err != nil {
				t.Fatalf("precondition: SetVisibility(%s): %v", slug, err)
			}
		}
	}
	put("published-card", catalog.VisibilityPublic)
	put("queued-card", "")
	put("hidden-card", catalog.VisibilityPrivate)
	put("removed-card", catalog.VisibilityPublic)
	if err := cm.DeleteEntry(ctx, "removed-card"); err != nil {
		t.Fatalf("precondition: DeleteEntry(removed-card): %v", err)
	}
	for i := 0; i < models.MaxPublicEntryLimit+5; i++ {
		put(fmt.Sprintf("bulk-%03d", i), catalog.VisibilityPublic)
	}

	list := func(h http.Handler, path string) []map[string]any {
		t.Helper()
		rec := do(t, h, http.MethodGet, path, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s answered %d %s", path, rec.Code, rec.Body.String())
		}
		var rows []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
			t.Fatalf("GET %s answered a body that is not a list: %v", path, err)
		}
		return rows
	}
	if got := list(operator, "/entries?visibility=pending&limit=500"); len(got) != 1 || got[0]["slug"] != "queued-card" {
		t.Fatalf("precondition: the operator's pending listing found %v", got)
	}

	queries := []string{
		"visibility=pending", "visibility=private", "Visibility=pending", "VISIBILITY=private",
		"visibility=public&visibility=private", "visibility=private&visibility=public",
		"deleted=true", "Deleted=true", "includedeleted=true", "include_deleted=true", "all=true",
		"limit=100000", "Limit=100000", "LIMIT=100000", "limit=-1", "limit=0",
		"limit=1&limit=100000", "search=card&Visibility=private&Limit=100000",
	}
	for _, q := range queries {
		rec := do(t, anon, http.MethodGet, "/public/entries?"+q, "")
		if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
			t.Errorf("anonymous GET /public/entries?%s answered %d %s; a spelling of a filter key must either list public entries or refuse with 400", q, rec.Code, rec.Body.String())
			continue
		}
		if rec.Code == http.StatusBadRequest {
			continue
		}
		rows := list(anon, "/public/entries?"+q)
		if len(rows) > models.MaxPublicEntryLimit {
			t.Errorf("anonymous GET /public/entries?%s listed %d rows; the public listing is capped at %d whatever the limit key is spelled", q, len(rows), models.MaxPublicEntryLimit)
		}
		for _, r := range rows {
			switch r["slug"] {
			case "queued-card", "hidden-card", "removed-card":
				t.Errorf("anonymous GET /public/entries?%s listed %q; no spelling of a query key may widen the public listing past visibility = public and deleted = false", q, r["slug"])
			}
			if d, _ := r["doc"].(string); d != "" {
				t.Errorf("anonymous GET /public/entries?%s carried a doc on %q; a listing never carries Doc", q, r["slug"])
			}
		}
	}
}
