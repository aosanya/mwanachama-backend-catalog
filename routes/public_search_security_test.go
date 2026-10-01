package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestPublicSearchNeverWidensTheAnonymousBoardPastPublic(t *testing.T) {
	operator, anon, cm := municipalityWard(t)
	ctx := t.Context()

	for _, slug := range []string{"held-pending", "held-private", "on-the-board"} {
		if _, err := cm.UpsertEntry(ctx, catalog.Entry{Slug: slug, Name: "Item " + slug, Summary: "zqxmarker " + slug, Doc: `{"_item":{"ward":"Kibera North"}}`}); err != nil {
			t.Fatalf("precondition: UpsertEntry(%s): %v", slug, err)
		}
	}
	if _, err := cm.SetVisibility(ctx, "held-private", catalog.VisibilityPrivate); err != nil {
		t.Fatalf("precondition: SetVisibility(private): %v", err)
	}
	if _, err := cm.SetVisibility(ctx, "on-the-board", catalog.VisibilityPublic); err != nil {
		t.Fatalf("precondition: SetVisibility(public): %v", err)
	}

	everything := do(t, operator, "GET", "/entries?search=zqxmarker", "")
	var all []catalog.Entry
	if err := json.Unmarshal(everything.Body.Bytes(), &all); err != nil || everything.Code != http.StatusOK || len(all) != 3 {
		t.Fatalf("precondition: the operator's search answered %d %s (%v), want all three entries", everything.Code, everything.Body.String(), err)
	}

	for _, query := range []string{
		"search=zqxmarker",
		"search=zqxmarker&visibility=private",
		"search=zqxmarker&visibility=pending",
		"search=held",
		"search=%25",
		"search=_",
	} {
		rec := do(t, anon, "GET", "/public/entries?"+query, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("anonymous GET /public/entries?%s answered %d %s, want 200", query, rec.Code, rec.Body.String())
		}
		var cards []catalog.Entry
		if err := json.Unmarshal(rec.Body.Bytes(), &cards); err != nil {
			t.Fatalf("decode %s: %v (%s)", query, err, rec.Body.String())
		}
		for _, c := range cards {
			if c.Slug != "on-the-board" || c.Visibility != catalog.VisibilityPublic {
				t.Errorf("anonymous GET /public/entries?%s returned %q (%s); a search term must only narrow the public board, never reach a pending or private entry", query, c.Slug, c.Visibility)
			}
		}
	}
}
