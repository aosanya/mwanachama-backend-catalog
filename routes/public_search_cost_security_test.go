package routes_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAnonymousWildcardSearchCostsNoMoreThanAPlainOne(t *testing.T) {
	_, anon, cm := municipalityWard(t)
	ctx := t.Context()
	long := strings.Repeat("abcdefghij", 200)
	for i := 0; i < 500; i++ {
		slug := fmt.Sprintf("item-%d", i)
		if _, err := cm.UpsertEntry(ctx, catalog.Entry{Slug: slug, Name: long, Summary: long, Doc: `{"_item":{"ward":"Kibera North"}}`}); err != nil {
			t.Fatalf("precondition: UpsertEntry(%s): %v", slug, err)
		}
		if _, err := cm.SetVisibility(ctx, slug, catalog.VisibilityPublic); err != nil {
			t.Fatalf("precondition: SetVisibility(%s): %v", slug, err)
		}
	}
	if rec := do(t, anon, "GET", "/public/entries?search=abcdefghij", ""); rec.Code != http.StatusOK || len(rec.Body.String()) < 1000 {
		t.Fatalf("precondition: an anonymous search that matches every public card answered %d with %d bytes", rec.Code, len(rec.Body.String()))
	}

	timed := func(q string) time.Duration {
		best := time.Duration(1 << 62)
		for range 3 {
			start := time.Now()
			rec := do(t, anon, "GET", "/public/entries?search="+url.QueryEscape(q), "")
			if rec.Code != http.StatusOK {
				t.Fatalf("anonymous search %d bytes answered %d %s", len(q), rec.Code, rec.Body.String())
			}
			best = min(best, time.Since(start))
		}
		return best
	}
	plain := timed("zzz")
	for _, q := range []string{strings.Repeat("%a", 2000) + "%z", strings.Repeat("_", 1990) + "z", strings.Repeat("%_", 2000) + "z"} {
		if got := timed(q); got > 20*plain+50*time.Millisecond {
			t.Errorf("an anonymous GET /public/entries?search= of %d wildcard-heavy bytes took %v against %v for a plain miss; want it within 20x — the one listing anonymous callers reach must not let a pattern multiply the store's work", len(q), got, plain)
		}
	}
}
