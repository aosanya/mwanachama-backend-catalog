package routes_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAnonymousSearchNeverReachesAnUnpublishedOrDeletedEntry(t *testing.T) {
	operator, anon, cm := municipalityWard(t)
	ctx := t.Context()
	doc := `{"_item":{"ward":"Kibera North"}}`
	put := func(slug, name, summary string) {
		t.Helper()
		if _, err := cm.UpsertEntry(ctx, catalog.Entry{Slug: slug, Name: name, Summary: summary, Doc: doc}); err != nil {
			t.Fatalf("precondition: UpsertEntry(%s): %v", slug, err)
		}
	}
	put("open-pothole", "Open pothole", "on the main road")
	if _, err := cm.SetVisibility(ctx, "open-pothole", catalog.VisibilityPublic); err != nil {
		t.Fatalf("precondition: SetVisibility(open-pothole): %v", err)
	}
	put("queued-report", "queuedmarker name", "queuedmarker summary")
	put("hidden-report", "hiddenmarker name", "hiddenmarker summary")
	if _, err := cm.SetVisibility(ctx, "hidden-report", catalog.VisibilityPrivate); err != nil {
		t.Fatalf("precondition: SetVisibility(hidden-report): %v", err)
	}
	put("removed-report", "removedmarker name", "removedmarker summary")
	if _, err := cm.SetVisibility(ctx, "removed-report", catalog.VisibilityPublic); err != nil {
		t.Fatalf("precondition: SetVisibility(removed-report): %v", err)
	}
	if err := cm.DeleteEntry(ctx, "removed-report", ""); err != nil {
		t.Fatalf("precondition: DeleteEntry(removed-report): %v", err)
	}

	slugs := func(h http.Handler, path string) []string {
		t.Helper()
		rec := do(t, h, http.MethodGet, path, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s answered %d %s", path, rec.Code, rec.Body.String())
		}
		var rows []struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
			t.Fatalf("GET %s answered a body that is not a list: %v", path, err)
		}
		out := make([]string, 0, len(rows))
		for _, r := range rows {
			out = append(out, r.Slug)
		}
		return out
	}

	if got := slugs(operator, "/entries?search=queuedmarker"); len(got) != 1 || got[0] != "queued-report" {
		t.Fatalf("precondition: the operator's search for the queued report found %v", got)
	}
	if got := slugs(anon, "/public/entries?search=pothole"); len(got) != 1 || got[0] != "open-pothole" {
		t.Fatalf("precondition: an anonymous search for the published card found %v", got)
	}

	for _, term := range []string{"queuedmarker", "hiddenmarker", "removedmarker", "marker", "name", "summary"} {
		for _, extra := range []string{"", "&visibility=pending", "&visibility=private", "&visibility="} {
			path := "/public/entries?search=" + url.QueryEscape(term) + extra
			for _, slug := range slugs(anon, path) {
				if slug != "open-pothole" {
					t.Errorf("anonymous GET %s listed %q; a search term matching an unpublished or deleted entry's name or summary must not widen the public listing past visibility = public and deleted = false", path, slug)
				}
			}
		}
	}
}
