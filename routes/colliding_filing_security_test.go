package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAnonymousFilingOnATakenSlugNeitherRevealsNorRewritesTheFirstReport(t *testing.T) {
	operator, anon, _ := municipalityWard(t)

	first := do(t, anon, "POST", "/suggestions",
		`{"slug":"broken-pipe","title":"Burst main by house 14","why":"ring 0712000000","proposal":"{\"_item\":{\"ward\":\"Kibera North\",\"reporter\":\"Jane W\"}}"}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("precondition: the first resident's report answered %d %s", first.Code, first.Body.String())
	}
	var original catalog.Suggestion
	if err := json.Unmarshal(first.Body.Bytes(), &original); err != nil || original.ID == "" {
		t.Fatalf("precondition: the first report's answer did not decode to a stored suggestion: %v %s", err, first.Body.String())
	}

	second := do(t, anon, "POST", "/suggestions",
		`{"slug":"broken-pipe","title":"overwritten","why":"overwritten","proposal":"{\"_item\":{\"ward\":\"elsewhere\"}}"}`)
	for _, secret := range []string{original.ID, "Burst main by house 14", "0712000000", "Jane W", "Kibera North"} {
		if strings.Contains(second.Body.String(), secret) {
			t.Errorf("a second anonymous filing on a taken slug answered %d %s, carrying %q from the first resident's report; want nothing of another filer's report in the answer",
				second.Code, second.Body.String(), secret)
		}
	}

	queue := do(t, operator, "GET", "/suggestions", "")
	var rows []catalog.Suggestion
	if err := json.Unmarshal(queue.Body.Bytes(), &rows); err != nil || queue.Code != http.StatusOK {
		t.Fatalf("precondition: the moderator queue answered %d %s (%v)", queue.Code, queue.Body.String(), err)
	}
	var kept catalog.Suggestion
	for _, r := range rows {
		if r.ID == original.ID {
			kept = r
		}
	}
	if kept.Title != original.Title || kept.Why != original.Why || kept.Proposal != original.Proposal {
		t.Errorf("after a second anonymous filing on %q the first report reads title=%q why=%q proposal=%q; want it untouched: title=%q why=%q proposal=%q",
			original.Slug, kept.Title, kept.Why, kept.Proposal, original.Title, original.Why, original.Proposal)
	}
}
