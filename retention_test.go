package mwanachamacatalog_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func createRetained(t *testing.T, cm catalog.CatalogManager, slug, disposeAt string) {
	t.Helper()
	ctx := t.Context()
	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: slug, Name: "Entry " + slug, Doc: doc, DisposeAt: disposeAt}); err != nil {
		t.Fatalf("create %s: %v", slug, err)
	}
	if _, err := cm.SetVisibility(ctx, slug, catalog.VisibilityPublic); err != nil {
		t.Fatalf("publish %s: %v", slug, err)
	}
}

func TestDisposeExpired_DisposesOnlyWhatHasRunOutAsTheClockSteps(t *testing.T) {
	cm, ctx := newManager(t)
	start := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	createRetained(t, cm, "a-month", start.Add(30*24*time.Hour).Format(time.RFC3339))
	createRetained(t, cm, "a-year", start.Add(365*24*time.Hour).Format(time.RFC3339))
	createRetained(t, cm, "kept", "")

	steps := []struct {
		at   time.Time
		want []string
	}{
		{start, []string{}},
		{start.Add(30*24*time.Hour - time.Second), []string{}},
		{start.Add(30 * 24 * time.Hour), []string{"a-month"}},
		{start.Add(31 * 24 * time.Hour), []string{}},
		{start.Add(400 * 24 * time.Hour), []string{"a-year"}},
		{start.Add(4000 * 24 * time.Hour), []string{}},
	}
	for _, s := range steps {
		got, err := cm.DisposeExpired(ctx, s.at)
		if err != nil {
			t.Fatalf("sweep at %s: %v", s.at, err)
		}
		if !reflect.DeepEqual(got, s.want) {
			t.Fatalf("sweep at %s disposed %v, want %v", s.at, got, s.want)
		}
	}

	for _, slug := range []string{"a-month", "a-year"} {
		if _, err := cm.OpenEntry(ctx, slug, ""); !errors.Is(err, catalog.ErrEntryNotFound) {
			t.Fatalf("open disposed %s = %v, want ErrEntryNotFound", slug, err)
		}
	}
	if _, err := cm.OpenEntry(ctx, "kept", ""); err != nil {
		t.Fatalf("open kept: %v", err)
	}
}

func TestDisposeExpired_ADisposedPrivateEntryNoLongerOpensWithItsKey(t *testing.T) {
	cm, ctx := newManager(t)
	createRetained(t, cm, "secret", "2030-01-01T00:00:00Z")
	if _, err := cm.SetVisibility(ctx, "secret", catalog.VisibilityPrivate); err != nil {
		t.Fatalf("private: %v", err)
	}
	_, raw, err := cm.IssueShareLink(ctx, "secret", "pilot", "operator")
	if err != nil {
		t.Fatalf("share: %v", err)
	}
	if _, err := cm.DisposeExpired(ctx, time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if _, err := cm.OpenEntry(ctx, "secret", raw); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("open disposed private with its key = %v, want ErrEntryNotFound", err)
	}
}

func TestDisposeAt_IsNormalisedToUTCAndRefusedWhenUnparseable(t *testing.T) {
	cm, ctx := newManager(t)
	e, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "x", Name: "X", Doc: doc, DisposeAt: "2030-01-01T03:00:00+03:00"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if e.DisposeAt != "2030-01-01T00:00:00Z" {
		t.Fatalf("dispose_at = %q, want UTC", e.DisposeAt)
	}
	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "y", Name: "Y", Doc: doc, DisposeAt: "next year"}); !errors.Is(err, catalog.ErrInvalidEntry) {
		t.Fatalf("unparseable dispose_at = %v, want ErrInvalidEntry", err)
	}
}
