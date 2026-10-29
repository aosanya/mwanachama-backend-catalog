package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/models"
)

func TestCAT29_OpenHole_AnonymousFilerSetsItsOwnQueuePriorityAndSource(t *testing.T) {
	operator, anon, _ := municipalityWard(t)

	honest := do(t, anon, "POST", "/suggestions", wardReportBody("honest-drain", "Kibera North", "waste", "counter"))
	if honest.Code != http.StatusCreated {
		t.Fatalf("precondition: an anonymous resident filing a plain report answered %d %s", honest.Code, honest.Body.String())
	}
	forged := do(t, anon, "POST", "/suggestions",
		`{"slug":"me-first","title":"Me first","priority":-2147483648,"source":"owner","run_ref":"run-forged"}`)
	if forged.Code != http.StatusCreated {
		t.Fatalf("precondition: an anonymous report carrying priority/source/run_ref answered %d %s", forged.Code, forged.Body.String())
	}

	queue := do(t, operator, "GET", "/suggestions", "")
	var rows []catalog.Suggestion
	if err := json.Unmarshal(queue.Body.Bytes(), &rows); err != nil || queue.Code != http.StatusOK || len(rows) != 2 {
		t.Fatalf("precondition: the moderator queue answered %d %s (%v), want both reports", queue.Code, queue.Body.String(), err)
	}
	var got catalog.Suggestion
	for _, r := range rows {
		if r.Slug == "me-first" {
			got = r
		}
	}
	if got.Priority != models.DefaultSuggestionPriority || got.Source == catalog.SuggestionSourceOwner || got.RunRef != "" || rows[0].Slug == "me-first" {
		t.Errorf("CAT29 (P1): an anonymous POST /suggestions stored priority=%d source=%q run_ref=%q and sits at queue position %q; "+
			"want priority=%d, source not %q, run_ref empty, and no queue-jump — on a mount that opens catalog.suggestion.file to the public these are the moderator's fields, not the filer's",
			got.Priority, got.Source, got.RunRef, rows[0].Slug, models.DefaultSuggestionPriority, catalog.SuggestionSourceOwner)
	}
}

func TestCAT30_OpenHole_AnonymousFilingTellsAHiddenBoardItemFromAMissingSlug(t *testing.T) {
	_, anon, cm := municipalityWard(t)
	ctx := t.Context()

	for _, slug := range []string{"held-pending", "names-a-person"} {
		if _, err := cm.UpsertEntry(ctx, catalog.Entry{Slug: slug, Name: "Item", Summary: "s", Doc: `{"_item":{"ward":"Kibera North"}}`}); err != nil {
			t.Fatalf("precondition: UpsertEntry(%s): %v", slug, err)
		}
	}
	if _, err := cm.SetVisibility(ctx, "names-a-person", catalog.VisibilityPrivate); err != nil {
		t.Fatalf("precondition: SetVisibility(private): %v", err)
	}
	for _, slug := range []string{"held-pending", "names-a-person"} {
		if rec := do(t, anon, "GET", "/entries/"+slug, ""); rec.Code != http.StatusNotFound {
			t.Fatalf("precondition: an anonymous open of hidden %q answered %d, want the 404 a missing slug gets", slug, rec.Code)
		}
	}

	missing := do(t, anon, "POST", "/suggestions", `{"slug":"never-existed","title":"t"}`)
	for _, slug := range []string{"held-pending", "names-a-person"} {
		hidden := do(t, anon, "POST", "/suggestions", `{"slug":"`+slug+`","title":"t"}`)
		if hidden.Code != missing.Code {
			t.Errorf("CAT30 (P1): an anonymous POST /suggestions naming hidden board item %q answered %d %s, a missing slug answered %d — "+
				"want the same answer for both; the difference tells anyone which unmoderated or private items exist, which GET /entries/{slug} deliberately hides",
				slug, hidden.Code, hidden.Body.String(), missing.Code)
		}
	}
}
