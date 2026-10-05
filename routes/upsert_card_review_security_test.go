package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCAT42_OpenHole_ResendWithANewNameOrSummaryKeepsTheApproval(t *testing.T) {
	operator, anon, _ := municipalityWard(t)
	doc := `{\"_item\":{\"ward\":\"Kibera North\",\"theme\":\"waste\"}}`
	reviewed := `{"name":"Blocked drain at Ngong Road","summary":"Reported by residents","doc":"` + doc + `"}`
	rewritten := `{"name":"Councillor took the drain money","summary":"Ward office is corrupt","doc":"` + doc + `"}`

	if rec := do(t, operator, http.MethodPut, "/entries/blocked-drain", reviewed); rec.Code != http.StatusOK {
		t.Fatalf("precondition: PUT /entries/blocked-drain answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, operator, http.MethodPost, "/entries/blocked-drain/visibility", `{"visibility":"public"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: approving blocked-drain answered %d %s", rec.Code, rec.Body.String())
	}

	type card struct {
		Slug       string `json:"slug"`
		Name       string `json:"name"`
		Summary    string `json:"summary"`
		Visibility string `json:"visibility"`
		ApprovedAt string `json:"approved_at"`
	}
	publicCards := func() []card {
		t.Helper()
		rec := do(t, anon, http.MethodGet, "/public/entries", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("anonymous GET /public/entries answered %d %s", rec.Code, rec.Body.String())
		}
		var out []card
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("anonymous GET /public/entries answered a body that is not a list: %v", err)
		}
		return out
	}
	if got := publicCards(); len(got) != 1 || got[0].Name != "Blocked drain at Ngong Road" {
		t.Fatalf("precondition: the approved card should be the one anonymous listing row; got %+v", got)
	}

	rec := do(t, operator, http.MethodPut, "/entries/blocked-drain", rewritten)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /entries/blocked-drain with a new name and summary answered %d %s", rec.Code, rec.Body.String())
	}
	var after card
	if err := json.Unmarshal(rec.Body.Bytes(), &after); err != nil {
		t.Fatalf("PUT answered a body that is not an entry: %v", err)
	}
	if after.Visibility == "public" || after.ApprovedAt != "" {
		t.Errorf("CAT42 (P1): re-sending an unchanged document with a new name and summary kept visibility %q and approved_at %q; want the entry back in pending with approved_at cleared, because the name and summary are the card an anonymous reader sees and the reviewer never saw this one", after.Visibility, after.ApprovedAt)
	}
	for _, c := range publicCards() {
		if c.Name != "Blocked drain at Ngong Road" || c.Summary != "Reported by residents" {
			t.Errorf("CAT42 (P1): anonymous GET /public/entries now lists %q with name %q and summary %q, written after the approval and never reviewed; want the listing to carry only reviewed text", c.Slug, c.Name, c.Summary)
		}
	}
}
