package mwanachamacatalog

import (
	"context"
	"errors"
	"testing"

	"github.com/aosanya/mwanachama-backend-catalog/models"
)

func TestWidenedVisibilityStaysClosedToAnonymousReaders(t *testing.T) {
	const embargoed = models.Visibility("embargoed")
	cm := managerOn(t, withExtraVisibility(t, string(embargoed)))
	ctx := context.Background()
	mustCreate(t, cm, ctx)

	e, err := cm.SetVisibility(ctx, "one-entry", embargoed)
	if err != nil || e.Visibility != embargoed {
		t.Fatalf("precondition: a domain-declared fourth visibility must be settable, got %q, err %v", e.Visibility, err)
	}

	if _, err := cm.OpenEntry(ctx, "one-entry", ""); !errors.Is(err, ErrEntryNotFound) {
		t.Errorf("an entry under a visibility the module does not know opened for a keyless reader: err = %v, want ErrEntryNotFound", err)
	}
	if _, err := cm.OpenEntry(ctx, "one-entry", "any-key"); !errors.Is(err, ErrEntryNotFound) {
		t.Errorf("an entry under a visibility the module does not know opened for a keyed reader: err = %v, want ErrEntryNotFound", err)
	}
	_, missing := cm.OpenEntry(ctx, "never-existed", "")
	_, widened := cm.OpenEntry(ctx, "one-entry", "")
	if missing == nil || widened == nil || missing.Error() != widened.Error() {
		t.Errorf("a widened-visibility slug answers differently from a missing one: %v vs %v", widened, missing)
	}

	public, err := cm.ListPublicEntries(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("list public: %v", err)
	}
	for _, p := range public {
		if p.Slug == "one-entry" {
			t.Errorf("the public listing carries an entry under visibility %q", embargoed)
		}
	}

	if _, _, err := cm.IssueShareLink(ctx, "one-entry", "label", "someone"); err == nil {
		t.Errorf("a share key was issued for an entry under visibility %q, which is not private", embargoed)
	}
}
