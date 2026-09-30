package mwanachamacatalog_test

import (
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT16_OpenBug_SearchTreatsUnderscoreAndPercentAsWildcards(t *testing.T) {
	cm, ctx := newManager(t)

	seedNamed := func(slug, name string) {
		if _, err := cm.CreateEntry(ctx, catalog.Entry{
			Slug: slug, Name: name, Summary: "a summary", Doc: doc,
		}); err != nil {
			t.Fatalf("create %s: %v", slug, err)
		}
	}
	seedNamed("s0", "Alpha_Beta")
	seedNamed("s1", "AlphaXBeta")
	seedNamed("s2", "Gamma")

	exact, err := cm.ListEntries(ctx, catalog.ListFilter{Search: "Alpha_Beta"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(exact) != 1 {
		t.Errorf("CAT16 open bug: ListEntries(Search: %q) matched %d entr(y/ies); a caller searching for the literal substring %q should match only %q, not treat '_' as a single-character SQL wildcard and also match %q",
			"Alpha_Beta", len(exact), "Alpha_Beta", "Alpha_Beta", "AlphaXBeta")
	}

	wide, err := cm.ListEntries(ctx, catalog.ListFilter{Search: "_"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(wide) >= 3 {
		t.Errorf("CAT16 open bug: ListEntries(Search: \"_\") returned all %d seeded entries; a one-character search term should narrow the list, not widen it to everything because '_' is an unescaped SQL LIKE wildcard", len(wide))
	}
}
