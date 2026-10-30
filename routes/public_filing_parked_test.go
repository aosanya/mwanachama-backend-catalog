package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAnonymousFilingOnATakenSlugIsParkedForTheModeratorUnderAnotherSlug(t *testing.T) {
	operator, anon, cm := municipalityWard(t)
	if _, err := cm.UpsertEntry(t.Context(), catalog.Entry{Slug: "held-pending", Name: "Item", Summary: "s", Doc: `{"_item":{"ward":"Kibera North"}}`}); err != nil {
		t.Fatalf("precondition: UpsertEntry: %v", err)
	}

	filed := do(t, anon, "POST", "/suggestions", `{"slug":"held-pending","title":"second sighting"}`)
	if filed.Code != http.StatusCreated {
		t.Fatalf("anonymous filing on a hidden entry's slug answered %d %s, want 201", filed.Code, filed.Body.String())
	}
	var answer catalog.Suggestion
	if err := json.Unmarshal(filed.Body.Bytes(), &answer); err != nil || answer.Slug != "held-pending" {
		t.Fatalf("answer slug = %q (%v), want the slug the filer asked for", answer.Slug, err)
	}

	queue := do(t, operator, "GET", "/suggestions", "")
	var rows []catalog.Suggestion
	if err := json.Unmarshal(queue.Body.Bytes(), &rows); err != nil || len(rows) != 1 {
		t.Fatalf("moderator queue answered %d %s (%v), want the parked report", queue.Code, queue.Body.String(), err)
	}
	if rows[0].ID != answer.ID || rows[0].Slug == "held-pending" || rows[0].Title != "second sighting" {
		t.Errorf("parked report = %+v, want id %s, a slug other than held-pending, title kept", rows[0], answer.ID)
	}
}

func TestOperatorFilingOnATakenSlugStillAnswersConflict(t *testing.T) {
	operator, _, _ := municipalityWard(t)
	if rec := do(t, operator, "POST", "/suggestions", `{"slug":"twice","title":"t"}`); rec.Code != http.StatusCreated {
		t.Fatalf("first filing answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, operator, "POST", "/suggestions", `{"slug":"twice","title":"t"}`); rec.Code != http.StatusConflict {
		t.Errorf("second filing by the operator answered %d %s, want 409", rec.Code, rec.Body.String())
	}
}
