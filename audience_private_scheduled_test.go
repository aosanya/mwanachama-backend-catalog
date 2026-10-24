package mwanachamacatalog_test

import (
	"errors"
	"testing"
	"time"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func slugsOf(list []catalog.Entry) map[string]bool {
	got := map[string]bool{}
	for _, e := range list {
		got[e.Slug] = true
	}
	return got
}

func TestBelowStaffAPrivateEntryOpensOnlyThroughItsKeyAndIsNeverListed(t *testing.T) {
	cm, f, ctx := fenced(t, catalog.AudienceMembers)
	seed(t, cm, ctx, "public-one", catalog.VisibilityPublic)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)
	_, key, err := cm.IssueShareLink(ctx, "private-one", "a link", "staff-1")
	if err != nil {
		t.Fatalf("issue share link: %v", err)
	}
	member := as(ctx, catalog.StandingMember)

	_, missingGet := f.GetEntry(member, "no-such-entry")
	_, privateGet := f.GetEntry(member, "private-one")
	if !errors.Is(privateGet, catalog.ErrEntryNotFound) || privateGet.Error() != missingGet.Error() {
		t.Fatalf("member get of private: want %v, got %v", missingGet, privateGet)
	}
	_, missingOpen := f.OpenEntry(member, "no-such-entry", "")
	for _, wrong := range []string{"", "not-the-key"} {
		_, privateOpen := f.OpenEntry(member, "private-one", wrong)
		if !errors.Is(privateOpen, catalog.ErrEntryNotFound) || privateOpen.Error() != missingOpen.Error() {
			t.Fatalf("member open of private with key %q: want %v, got %v", wrong, missingOpen, privateOpen)
		}
	}
	if e, err := f.OpenEntry(member, "private-one", key); err != nil || e.Slug != "private-one" {
		t.Fatalf("member open of private with its key: got %+v, err %v", e, err)
	}

	for _, filter := range []catalog.ListFilter{{}, {Visibility: catalog.VisibilityPrivate}} {
		list, err := f.ListEntries(member, filter)
		if err != nil {
			t.Fatal(err)
		}
		if got := slugsOf(list); got["private-one"] {
			t.Fatalf("member list %+v carried the private entry: %v", filter, got)
		}
	}
	list, err := f.ListEntries(member, catalog.ListFilter{})
	if err != nil || !slugsOf(list)["public-one"] {
		t.Fatalf("member list: want public-one, got %+v, err %v", list, err)
	}
}

func TestBelowStaffAScheduledEntryAnswersExactlyAsAMissingSlug(t *testing.T) {
	cm, f, ctx := fenced(t, catalog.AudienceMembers)
	createScheduled(t, cm, "later", time.Now().Add(time.Hour).Format(time.RFC3339))
	createScheduled(t, cm, "earlier", time.Now().Add(-time.Hour).Format(time.RFC3339))
	member := as(ctx, catalog.StandingMember)

	_, missing := f.GetEntry(member, "no-such-entry")
	_, later := f.GetEntry(member, "later")
	if !errors.Is(later, catalog.ErrEntryNotFound) || later.Error() != missing.Error() {
		t.Fatalf("member get of a scheduled entry: want %v, got %v", missing, later)
	}
	if _, err := f.OpenEntry(member, "later", ""); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("member open of a scheduled entry: want ErrEntryNotFound, got %v", err)
	}
	if _, err := f.GetEntry(member, "earlier"); err != nil {
		t.Fatalf("member get of a published entry: %v", err)
	}

	list, err := f.ListEntries(member, catalog.ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if got := slugsOf(list); got["later"] || !got["earlier"] {
		t.Fatalf("member list: want earlier only, got %v", got)
	}
	list, err = f.ListEntries(member, catalog.ListFilter{Limit: 1})
	if err != nil || len(list) != 1 || list[0].Slug != "earlier" {
		t.Fatalf("member list with limit 1: want earlier, got %+v, err %v", list, err)
	}
}

func TestStaffStillReadPrivateAndScheduledEntries(t *testing.T) {
	cm, f, ctx := fenced(t, catalog.AudienceMembers)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)
	createScheduled(t, cm, "later", time.Now().Add(time.Hour).Format(time.RFC3339))
	staff := as(ctx, catalog.StandingStaff)

	for _, slug := range []string{"private-one", "later"} {
		if _, err := f.GetEntry(staff, slug); err != nil {
			t.Fatalf("staff get of %s: %v", slug, err)
		}
	}
	list, err := f.ListEntries(staff, catalog.ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if got := slugsOf(list); !got["private-one"] || !got["later"] {
		t.Fatalf("staff list: want both, got %v", got)
	}
}
