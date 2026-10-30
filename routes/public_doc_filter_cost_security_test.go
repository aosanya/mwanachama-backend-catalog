package routes_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT41_AnonymousListingRefusesStackedDocFilters(t *testing.T) {
	_, anon, cm := municipalityWard(t)
	ctx := t.Context()
	if _, err := cm.UpsertEntry(ctx, catalog.Entry{Slug: "item-1", Name: "Pothole", Summary: "s", Doc: `{"_item":{"ward":"Kibera North"}}`}); err != nil {
		t.Fatalf("precondition: UpsertEntry: %v", err)
	}
	if _, err := cm.SetVisibility(ctx, "item-1", catalog.VisibilityPublic); err != nil {
		t.Fatalf("precondition: SetVisibility: %v", err)
	}
	filter := "docs=" + url.QueryEscape(`{"path":"_item.ward","value":"Kibera North"}`)

	timed := func(n int) (time.Duration, int, string) {
		q := strings.TrimSuffix(strings.Repeat(filter+"&", n), "&")
		best := time.Duration(1 << 62)
		var code int
		var body string
		for range 2 {
			start := time.Now()
			rec := do(t, anon, "GET", "/public/entries?"+q, "")
			best = min(best, time.Since(start))
			code, body = rec.Code, rec.Body.String()
		}
		return best, code, body
	}

	one, code, body := timed(1)
	if code != http.StatusOK || !strings.Contains(body, `"item-1"`) {
		t.Fatalf("precondition: an anonymous listing with one document filter answered %d %s", code, body)
	}

	const many = 13000
	got, code, body := timed(many)
	if code >= 500 {
		t.Errorf("CAT41 (P2): an anonymous GET /public/entries carrying %d document filters (a %d-byte query, inside the default 1 MiB header limit) answered %d %s; want a 400 refusing the filter count before the store is asked", many, many*len(filter), code, body)
	}
	if got > 20*one+50*time.Millisecond {
		t.Errorf("CAT41 (P2): an anonymous GET /public/entries carrying %d document filters took %v against %v for one filter; want it within 20x — the one listing anonymous callers reach must cap how many filters a request may stack", many, got, one)
	}
}
