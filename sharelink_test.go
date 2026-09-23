package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/models"
)

// The raw key is handed back once and never stored, so a reader of the
// store cannot recover it.
func TestIssueShareLink_StoresOnlyTheHash(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	link, raw, err := cm.IssueShareLink(ctx, "private-one", "pilot", "owner")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if raw == "" || len(raw) != 64 {
		t.Fatalf("raw key = %q, want 64 hex characters", raw)
	}
	if link.KeyHash == raw {
		t.Fatal("the stored hash is the raw key")
	}
	if link.KeyHash != models.HashKey(raw) {
		t.Errorf("KeyHash = %q, want the hash of the raw key", link.KeyHash)
	}

	stored, err := cm.ListShareLinks(ctx, "private-one")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if stored[0].KeyHash == raw {
		t.Error("the raw key reached the store")
	}
}

// Rotating issues a second key without killing the first, so a handover can
// overlap; retiring is the separate, deliberate act.
func TestIssueShareLink_RotationKeepsBothUntilRetired(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	first, rawFirst, err := cm.IssueShareLink(ctx, "private-one", "first", "owner")
	if err != nil {
		t.Fatalf("issue first: %v", err)
	}
	_, rawSecond, err := cm.IssueShareLink(ctx, "private-one", "second", "owner")
	if err != nil {
		t.Fatalf("issue second: %v", err)
	}

	for _, k := range []string{rawFirst, rawSecond} {
		if _, err := cm.OpenEntry(ctx, "private-one", k); err != nil {
			t.Fatalf("both keys should open: %v", err)
		}
	}

	if _, err := cm.RetireShareLink(ctx, first.ID); err != nil {
		t.Fatalf("retire: %v", err)
	}
	if _, err := cm.OpenEntry(ctx, "private-one", rawFirst); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Errorf("retired key: err = %v, want ErrEntryNotFound", err)
	}
	if _, err := cm.OpenEntry(ctx, "private-one", rawSecond); err != nil {
		t.Errorf("live key should still open: %v", err)
	}
}

// A link against a public entry would imply a restriction that is not
// there; against a pending one it would outlive the review.
func TestIssueShareLink_OnlyForAPrivateEntry(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "public-one", catalog.VisibilityPublic)
	seed(t, cm, ctx, "pending-one", catalog.VisibilityPending)

	for _, slug := range []string{"public-one", "pending-one"} {
		if _, _, err := cm.IssueShareLink(ctx, slug, "", "owner"); !errors.Is(err, catalog.ErrNotPrivate) {
			t.Errorf("%s: err = %v, want ErrNotPrivate", slug, err)
		}
	}
}

func TestIssueShareLink_EveryKeyIsDifferent(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		_, raw, err := cm.IssueShareLink(ctx, "private-one", "", "owner")
		if err != nil {
			t.Fatalf("issue %d: %v", i, err)
		}
		if seen[raw] {
			t.Fatalf("key %q was issued twice", raw)
		}
		seen[raw] = true
	}
}

func TestRetireShareLink_UnknownID(t *testing.T) {
	cm, ctx := newManager(t)
	if _, err := cm.RetireShareLink(ctx, "no-such-id"); !errors.Is(err, catalog.ErrShareLinkNotFound) {
		t.Fatalf("err = %v, want ErrShareLinkNotFound", err)
	}
}
