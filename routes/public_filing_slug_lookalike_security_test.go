package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"unicode/utf8"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAnAnonymousFilingWithALookAlikeSlugIsRefusedAndAnInvalidTitleIsStoredAsValidText(t *testing.T) {
	operator, anon, _ := municipalityWard(t)

	if rec := do(t, anon, "POST", "/suggestions", wardReportBody("drain", "Kibera North", "waste", "counter")); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: an anonymous report on slug drain answered %d %s", rec.Code, rec.Body.String())
	}

	for _, slug := range []string{"DRAIN", "drain\n", "drain​", "дrain", "ｄrain", "dràin", "drain\xff", "drain/../x"} {
		body := `{"slug":` + quotedSlug(t, slug) + `,"title":"t","why":"w"}`
		if slug == "drain\xff" {
			body = "{\"slug\":\"drain\xff\",\"title\":\"t\",\"why\":\"w\"}"
		}
		if rec := do(t, anon, "POST", "/suggestions", body); rec.Code != http.StatusBadRequest {
			t.Errorf("an anonymous report on look-alike slug %q answered %d %s, want 400 so no second report reads like drain", slug, rec.Code, rec.Body.String())
		}
	}

	if rec := do(t, anon, "POST", "/suggestions", "{\"slug\":\"bad-bytes\",\"title\":\"Blocked \xff\xfe drain\",\"why\":\"w\"}"); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: an anonymous report whose title carries invalid UTF-8 answered %d %s", rec.Code, rec.Body.String())
	}

	queue := do(t, operator, "GET", "/suggestions", "")
	var rows []catalog.Suggestion
	if err := json.Unmarshal(queue.Body.Bytes(), &rows); err != nil || queue.Code != http.StatusOK {
		t.Fatalf("the moderator queue answered %d %s (%v)", queue.Code, queue.Body.String(), err)
	}
	if len(rows) != 2 {
		t.Errorf("only drain and bad-bytes should be stored, the queue holds %d: %s", len(rows), queue.Body.String())
	}
	for _, r := range rows {
		if !utf8.ValidString(r.Slug) || !utf8.ValidString(r.Title) {
			t.Errorf("stored report %q carries invalid UTF-8 (title %q), which a Postgres text column refuses with a 500", r.Slug, r.Title)
		}
	}
}

func quotedSlug(t *testing.T, v string) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}
