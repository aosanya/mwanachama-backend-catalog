package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func nestedProposalBody(slug string, depth int) string {
	inner := strings.Repeat(`{"a":`, depth) + "1" + strings.Repeat("}", depth)
	return `{"slug":"` + slug + `","title":"t","proposal":"` + strings.ReplaceAll(inner, `"`, `\"`) + `"}`
}

func TestAnonymousDeeplyNestedProposalIsRefusedAndNeverBreaksTheModeratorQueue(t *testing.T) {
	operator, anon, _ := municipalityWard(t)

	shallow := do(t, anon, "POST", "/suggestions", nestedProposalBody("shallow-nest", 1000))
	if shallow.Code != http.StatusCreated {
		t.Fatalf("precondition: an anonymous filing with a 1,000-deep proposal answered %d %s, want 201", shallow.Code, shallow.Body.String())
	}

	tooDeep := do(t, anon, "POST", "/suggestions", nestedProposalBody("too-deep-nest", 10001))
	if tooDeep.Code != http.StatusBadRequest {
		t.Errorf("an anonymous filing with a 10,001-deep proposal answered %d %s, want 400 from the declared json_object rule before storage sees it",
			tooDeep.Code, tooDeep.Body.String())
	}

	do(t, anon, "POST", "/suggestions", nestedProposalBody("mid-nest", 4000))

	queue := do(t, operator, "GET", "/suggestions", "")
	var rows []catalog.Suggestion
	if err := json.Unmarshal(queue.Body.Bytes(), &rows); err != nil || queue.Code != http.StatusOK {
		t.Fatalf("after deep anonymous filings the moderator queue answered %d (%v), want a readable 200", queue.Code, err)
	}
	seen := map[string]bool{}
	for _, r := range rows {
		seen[r.Slug] = true
		var v map[string]any
		if err := json.Unmarshal([]byte(r.Proposal), &v); err != nil {
			t.Errorf("queued suggestion %q carries a proposal that no longer decodes as an object: %v", r.Slug, err)
		}
	}
	if !seen["shallow-nest"] || seen["too-deep-nest"] {
		t.Errorf("moderator queue holds %v, want shallow-nest present and too-deep-nest absent", seen)
	}
}
