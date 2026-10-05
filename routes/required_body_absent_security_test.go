package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestARequiredBodyArgumentLeftOutIsRefusedAndChangesNothing(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "alpha", catalog.VisibilityPublic)
	filed, err := cm.FileSuggestion(ctx, catalog.Suggestion{Slug: "beta", Title: "Beta", Why: "asked for", Proposal: "{}"})
	if err != nil {
		t.Fatalf("FileSuggestion: %v", err)
	}
	if _, err := cm.DecideSuggestion(ctx, filed.ID, catalog.SuggestionStatusApproved, 0); err != nil {
		t.Fatalf("DecideSuggestion: %v", err)
	}
	suggestion := "/suggestions/" + filed.ID

	read := func(path string, into any) {
		t.Helper()
		rec := do(t, h, http.MethodGet, path, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d %s", path, rec.Code, rec.Body.String())
		}
		if err := json.Unmarshal(rec.Body.Bytes(), into); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	var entry catalog.Entry
	read("/entries/alpha", &entry)
	if entry.Visibility != catalog.VisibilityPublic {
		t.Fatalf("precondition: alpha is %q, want public", entry.Visibility)
	}
	var before catalog.Suggestion
	read(suggestion, &before)
	if before.Status != catalog.SuggestionStatusApproved {
		t.Fatalf("precondition: suggestion is %q, want approved", before.Status)
	}

	for _, path := range []string{
		"/entries/alpha/visibility",
		suggestion + "/decide",
		suggestion + "/attempt",
		suggestion + "/fulfil",
		suggestion + "/comments",
		suggestion + "/matches",
		"/page-views",
	} {
		rec := do(t, h, http.MethodPost, path, `{}`)
		if rec.Code < 400 || rec.Code >= 500 {
			t.Errorf("POST %s {} = %d %s, want a 4xx refusal for the left-out required argument", path, rec.Code, rec.Body.String())
		}
	}

	read("/entries/alpha", &entry)
	if entry.Visibility != catalog.VisibilityPublic {
		t.Errorf("alpha after the refused calls is %q, want public untouched", entry.Visibility)
	}
	var after catalog.Suggestion
	read(suggestion, &after)
	if after.Status != before.Status || after.Attempts != before.Attempts || after.Priority != before.Priority {
		t.Errorf("suggestion after the refused calls = status %q attempts %d priority %d, want %q %d %d untouched",
			after.Status, after.Attempts, after.Priority, before.Status, before.Attempts, before.Priority)
	}
	if rec := do(t, h, http.MethodGet, suggestion+"/comments", ""); rec.Code != http.StatusOK || string(rec.Body.Bytes()) != "[]\n" {
		t.Errorf("comments after the refused calls = %d %q, want 200 []", rec.Code, rec.Body.String())
	}
}
