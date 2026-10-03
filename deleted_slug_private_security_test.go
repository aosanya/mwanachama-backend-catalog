package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/models"
)

func TestADeletedEntrysPrivateNotesNeverReachALaterEntryAtItsSlug(t *testing.T) {
	cm, ctx := newManager(t)

	if _, err := cm.UpsertEntry(ctx, catalog.Entry{Slug: "reused", Name: "First", Doc: `{"title":"first"}`}); err != nil {
		t.Fatalf("create first entry: %v", err)
	}
	if _, err := cm.SetVisibility(ctx, "reused", models.VisibilityPrivate); err != nil {
		t.Fatalf("make first entry private: %v", err)
	}
	if _, err := cm.PutPrivate(ctx, catalog.EntryPrivate{Slug: "reused", Readme: "first entry's notes"}); err != nil {
		t.Fatalf("write first entry's notes: %v", err)
	}
	if p, err := cm.GetPrivate(ctx, "reused"); err != nil || p.Readme != "first entry's notes" {
		t.Fatalf("precondition: first entry's notes not readable (%+v, %v)", p, err)
	}
	if err := cm.DeleteEntry(ctx, "reused"); err != nil {
		t.Fatalf("delete first entry: %v", err)
	}

	if p, err := cm.GetPrivate(ctx, "reused"); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Errorf("a deleted entry's notes answered %+v, %v; want ErrEntryNotFound", p, err)
	}

	second, err := cm.UpsertEntry(ctx, catalog.Entry{Slug: "reused", Name: "Second", Doc: `{"title":"second"}`})
	if err == nil {
		p, perr := cm.GetPrivate(ctx, "reused")
		if perr == nil && p.Readme != "" {
			t.Fatalf("a later entry %q at a deleted slug reads the deleted entry's notes %q; want none", second.Name, p.Readme)
		}
		return
	}
	if !errors.Is(err, catalog.ErrEntryExists) {
		t.Fatalf("re-creating a deleted slug: %v; want ErrEntryExists or a fresh entry with no notes", err)
	}
}
