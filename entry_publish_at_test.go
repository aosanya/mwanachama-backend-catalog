package mwanachamacatalog_test

import (
	"errors"
	"testing"
	"time"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func createScheduled(t *testing.T, cm catalog.CatalogManager, slug, at string) {
	t.Helper()
	ctx := t.Context()
	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: slug, Name: "Entry " + slug, Doc: doc, PublishAt: at}); err != nil {
		t.Fatalf("create %s: %v", slug, err)
	}
	if _, err := cm.SetVisibility(ctx, slug, catalog.VisibilityPublic); err != nil {
		t.Fatalf("publish %s: %v", slug, err)
	}
}

func TestPublishAt_AFutureEntryAnswersAsMissingOnEveryAnonymousRead(t *testing.T) {
	cm, ctx := newManager(t)
	createScheduled(t, cm, "later", time.Now().Add(time.Hour).Format(time.RFC3339))
	createScheduled(t, cm, "earlier", time.Now().Add(-time.Hour).Format(time.RFC3339))
	createScheduled(t, cm, "unscheduled", "")

	if _, err := cm.OpenEntry(ctx, "later", ""); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("open a future entry = %v, want ErrEntryNotFound", err)
	}
	if err := cm.RecordPageView(ctx, catalog.PageKind("entry"), "later"); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("count a view of a future entry = %v, want ErrEntryNotFound", err)
	}
	for _, slug := range []string{"earlier", "unscheduled"} {
		if _, err := cm.OpenEntry(ctx, slug, ""); err != nil {
			t.Fatalf("open %s: %v", slug, err)
		}
	}

	listed, err := cm.ListPublicEntries(ctx, catalog.ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, e := range listed {
		got[e.Slug] = true
	}
	if got["later"] || !got["earlier"] || !got["unscheduled"] {
		t.Fatalf("public listing = %v, want earlier and unscheduled only", got)
	}

	all, err := cm.ListEntries(ctx, catalog.ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("operator listing has %d entries, want all 3", len(all))
	}
}

func TestPublishAt_IsStoredInUTCAndRefusedWhenNotATime(t *testing.T) {
	cm, ctx := newManager(t)
	e, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "zoned", Name: "Zoned", Doc: doc, PublishAt: "2030-01-01T03:00:00+03:00"})
	if err != nil {
		t.Fatal(err)
	}
	if e.PublishAt != "2030-01-01T00:00:00Z" {
		t.Fatalf("publish_at = %q, want UTC", e.PublishAt)
	}
	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "bad", Name: "Bad", Doc: doc, PublishAt: "tomorrow"}); !errors.Is(err, catalog.ErrInvalidEntry) {
		t.Fatalf("create with publish_at tomorrow = %v, want ErrInvalidEntry", err)
	}
}
