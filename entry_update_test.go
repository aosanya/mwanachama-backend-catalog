package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT11_OpenBug_UpdateEntryBypassesVisibilityMachine(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	_, raw, err := cm.IssueShareLink(ctx, "private-one", "", "owner")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	existing, err := cm.GetEntry(ctx, "private-one")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	existing.Visibility = catalog.VisibilityPublic

	if _, err := cm.UpdateEntry(ctx, existing); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := cm.GetEntry(ctx, "private-one")
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if got.Visibility != catalog.VisibilityPrivate {
		t.Fatalf("CAT11: caller wrote Visibility=%q through UpdateEntry and it stuck (got %q); "+
			"visibility changes must only ever happen through SetVisibility, the way UpsertEntry "+
			"already refuses to accept a caller-supplied Visibility",
			catalog.VisibilityPublic, got.Visibility)
	}

	if _, err := cm.OpenEntry(ctx, "private-one", ""); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("CAT11: OpenEntry with no key = %v, want ErrEntryNotFound; UpdateEntry made a "+
			"private Entry openable by anyone without going through SetVisibility", err)
	}

	links, err := cm.ListShareLinks(ctx, "private-one")
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if len(links) != 1 || links[0].Status != catalog.LinkStatusActive {
		t.Fatalf("CAT11: share link = %+v, want it still active; CLAUDE.md's preserved behaviour "+
			"\"leaving private retires every live key\" must hold on every write path, not only "+
			"SetVisibility's", links)
	}
	if _, err := cm.OpenEntry(ctx, "private-one", raw); err != nil {
		t.Fatalf("CAT11: OpenEntry with the still-live key = %v, want it to still work", err)
	}
}
