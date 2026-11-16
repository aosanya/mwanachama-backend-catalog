package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

// A deleted Entry is invisible to every read this module offers — GetEntry,
// OpenEntry and ListEntries all answer as if the slug were never taken
// (TestDeleteEntry_HidesItFromEveryRead). CAT37: the write side disagrees.
// DeleteEntry only sets `deleted = true`; it never frees the DB-level
// `unique` constraint on `entry.slug`, so the old, invisible row keeps the
// slug forever and the one operation a mounting host exposes for creating
// an Entry — UpsertEntry, wired to `PUT /entries/{slug}` — fails on a slug
// every read already reports as free.
func TestCAT37_DeletedEntrySlugCanBeReused(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "ghost-slug", catalog.VisibilityPublic)

	if err := cm.DeleteEntry(ctx, "ghost-slug", ""); err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}

	if _, err := cm.GetEntry(ctx, "ghost-slug"); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("GetEntry after delete: err = %v, want ErrEntryNotFound", err)
	}

	got, err := cm.UpsertEntry(ctx, catalog.Entry{
		Slug:    "ghost-slug",
		Name:    "Totally Different New Owner",
		Summary: "unrelated to the deleted entry",
		Doc:     doc,
	})
	if err != nil {
		t.Fatalf(
			"CAT37 open bug: UpsertEntry on slug %q answered %v, want it to succeed — "+
				"GetEntry, OpenEntry and ListEntries all already told the caller this slug "+
				"was free (ErrEntryNotFound/empty list), so a slug a soft-deleted Entry once "+
				"held is sterilised forever with nothing on any read surface to explain why",
			"ghost-slug", err)
		return
	}
	if got.Name != "Totally Different New Owner" {
		t.Fatalf("CAT37: got.Name = %q, want the new entry's own name", got.Name)
	}
}
