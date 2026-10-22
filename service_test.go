package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAServiceIsAnEntryWhoseTermsReadBackAfterPublishing(t *testing.T) {
	cm, ctx := newManager(t)
	seeds := []struct {
		slug, name string
		terms      catalog.ServiceTerms
	}{
		{"business-permit", "Business permit", catalog.ServiceTerms{Fee: "KES 2,000, paid to the agency directly", Form: "form-1", Appointments: true}},
		{"raise-something", "Raise something", catalog.ServiceTerms{Opens: catalog.ServiceOpensRaise}},
	}
	for _, s := range seeds {
		e, err := catalog.ServiceEntry(s.slug, s.name, "what it is", s.terms)
		if err != nil {
			t.Fatalf("%s: %v", s.slug, err)
		}
		if _, err := cm.UpsertEntry(ctx, e); err != nil {
			t.Fatalf("%s: %v", s.slug, err)
		}
		if _, err := cm.SetVisibility(ctx, s.slug, catalog.VisibilityPublic); err != nil {
			t.Fatalf("publish %s: %v", s.slug, err)
		}
	}
	permit, err := cm.GetEntry(ctx, "business-permit")
	if err != nil {
		t.Fatal(err)
	}
	terms, err := catalog.ServiceTermsOf(permit)
	if err != nil || terms.Opens != catalog.ServiceOpensForm || terms.Form != "form-1" || !terms.Appointments || terms.Fee == "" {
		t.Fatalf("permit terms = %+v (%v)", terms, err)
	}
	raise, err := cm.GetEntry(ctx, "raise-something")
	if err != nil {
		t.Fatal(err)
	}
	terms, err = catalog.ServiceTermsOf(raise)
	if err != nil || terms.Opens != catalog.ServiceOpensRaise || terms.Form != "" || terms.Appointments {
		t.Fatalf("raise terms = %+v (%v)", terms, err)
	}
}

func TestAServiceWithNoWayToApplyIsRefused(t *testing.T) {
	for name, terms := range map[string]catalog.ServiceTerms{
		"form opening with no form":  {Fee: "free"},
		"raise opening with a form":  {Opens: catalog.ServiceOpensRaise, Form: "form-1"},
		"raise taking appointments":  {Opens: catalog.ServiceOpensRaise, Appointments: true},
		"an opening nobody declared": {Opens: "elsewhere", Form: "form-1"},
	} {
		if _, err := catalog.ServiceEntry("s", "S", "", terms); !errors.Is(err, catalog.ErrInvalidServiceTerms) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, doc := range map[string]string{
		"empty":       "",
		"not json":    "{",
		"not a terms": `{"opens":"form"}`,
	} {
		if _, err := catalog.ServiceTermsOf(catalog.Entry{Doc: doc}); !errors.Is(err, catalog.ErrInvalidServiceTerms) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestAServiceDefaultsToOpeningItsForm(t *testing.T) {
	terms, err := catalog.ServiceTermsOf(catalog.Entry{Doc: `{"form":"form-9","fee":"free"}`})
	if err != nil || terms.Opens != catalog.ServiceOpensForm || terms.Form != "form-9" {
		t.Fatalf("terms = %+v (%v)", terms, err)
	}
}

func TestAServicesMountChecksTermsOnEveryEntryWrite(t *testing.T) {
	base, ctx := newManager(t)
	cm := catalog.WithServiceTerms(base)
	good, err := catalog.ServiceEntry("permit", "Permit", "what it is", catalog.ServiceTerms{Form: "form-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.CreateEntry(ctx, good); err != nil {
		t.Fatalf("create: %v", err)
	}
	bad := []string{``, `[]`, `{"opens":"form"}`, `{"opens":"raise","form":"form-1"}`, `{"opens":"raise","appointments":true}`, `{"opens":"elsewhere","form":"form-1"}`}
	for _, doc := range bad {
		e := catalog.Entry{Slug: "permit", Name: "Permit", Doc: doc}
		if _, err := cm.UpdateEntry(ctx, e); !errors.Is(err, catalog.ErrInvalidServiceTerms) {
			t.Fatalf("update %q: %v", doc, err)
		}
		if _, err := cm.UpsertEntry(ctx, e); !errors.Is(err, catalog.ErrInvalidServiceTerms) {
			t.Fatalf("upsert %q: %v", doc, err)
		}
		e.Slug = "other"
		if _, err := cm.CreateEntry(ctx, e); !errors.Is(err, catalog.ErrInvalidServiceTerms) {
			t.Fatalf("create %q: %v", doc, err)
		}
	}
	stored, err := base.GetEntry(ctx, "permit")
	if err != nil {
		t.Fatal(err)
	}
	if terms, err := catalog.ServiceTermsOf(stored); err != nil || terms.Form != "form-1" {
		t.Fatalf("stored terms = %+v (%v)", terms, err)
	}
	if _, err := base.GetEntry(ctx, "other"); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("refused create stored: %v", err)
	}
	raise := catalog.Entry{Slug: "permit", Name: "Raise something", Visibility: catalog.VisibilityPending, Doc: `{"opens":"raise"}`}
	if _, err := cm.UpdateEntry(ctx, raise); err != nil {
		t.Fatalf("raise update: %v", err)
	}
}
