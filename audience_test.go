package mwanachamacatalog_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/routes"
)

type standingKey struct{}

func as(ctx context.Context, s catalog.Standing) context.Context {
	return context.WithValue(ctx, standingKey{}, s)
}

func standingFromContext(ctx context.Context) catalog.Standing {
	s, _ := ctx.Value(standingKey{}).(catalog.Standing)
	return s
}

func fenced(t *testing.T, a catalog.Audience) (catalog.CatalogManager, catalog.CatalogManager, context.Context) {
	t.Helper()
	cm, ctx := newManager(t)
	f, err := catalog.WithAudience(cm, a, standingFromContext)
	if err != nil {
		t.Fatalf("with audience: %v", err)
	}
	return cm, f, ctx
}

func TestAMembersOnlyCatalogueRefusesAnAnonymousReadAsNotFound(t *testing.T) {
	cm, f, ctx := fenced(t, catalog.AudienceMembers)
	seed(t, cm, ctx, "entry-one", catalog.VisibilityPublic)

	if _, err := f.OpenEntry(ctx, "entry-one", ""); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("anonymous open: want ErrEntryNotFound, got %v", err)
	}
	if _, err := f.OpenEntry(ctx, "no-such-entry", ""); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("missing slug: want ErrEntryNotFound, got %v", err)
	}
	list, err := f.ListPublicEntries(ctx, catalog.ListFilter{})
	if err != nil || len(list) != 0 {
		t.Fatalf("anonymous list: want nothing, got %d entries, err %v", len(list), err)
	}

	member := as(ctx, catalog.StandingMember)
	if _, err := f.OpenEntry(member, "entry-one", ""); err != nil {
		t.Fatalf("member open: %v", err)
	}
	list, err = f.ListPublicEntries(member, catalog.ListFilter{})
	if err != nil || len(list) != 1 {
		t.Fatalf("member list: want 1 entry, got %d, err %v", len(list), err)
	}
}

func TestAStaffOnlyCatalogueRefusesAMember(t *testing.T) {
	cm, f, ctx := fenced(t, catalog.AudienceStaff)
	seed(t, cm, ctx, "entry-one", catalog.VisibilityPublic)

	if _, err := f.OpenEntry(as(ctx, catalog.StandingMember), "entry-one", ""); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("member open of a staff catalogue: want ErrEntryNotFound, got %v", err)
	}
	if _, err := f.OpenEntry(as(ctx, catalog.StandingStaff), "entry-one", ""); err != nil {
		t.Fatalf("staff open: %v", err)
	}
}

func TestAPublicCatalogueOpensWithoutSigningIn(t *testing.T) {
	cm, f, ctx := fenced(t, catalog.AudiencePublic)
	seed(t, cm, ctx, "entry-one", catalog.VisibilityPublic)
	if _, err := f.OpenEntry(ctx, "entry-one", ""); err != nil {
		t.Fatalf("anonymous open of a public catalogue: %v", err)
	}
}

func TestAnUnknownAudienceIsRefused(t *testing.T) {
	cm, _ := newManager(t)
	if _, err := catalog.WithAudience(cm, "everyone", standingFromContext); err == nil {
		t.Fatal("want an error for an undeclared audience")
	}
}

func TestPrivatePartsNeverReachANonStaffRead(t *testing.T) {
	cm, f, ctx := fenced(t, catalog.AudiencePublic)
	seed(t, cm, ctx, "entry-one", catalog.VisibilityPublic)
	staff := as(ctx, catalog.StandingStaff)
	if _, err := f.PutPrivate(staff, catalog.EntryPrivate{Slug: "entry-one", Doc: `{"cost":40,"margin":10}`}); err != nil {
		t.Fatalf("staff put private: %v", err)
	}

	for _, who := range []catalog.Standing{catalog.StandingNone, catalog.StandingMember} {
		reader := as(ctx, who)
		if _, err := f.GetPrivate(reader, "entry-one"); !errors.Is(err, catalog.ErrEntryNotFound) {
			t.Fatalf("standing %d read private: want ErrEntryNotFound, got %v", who, err)
		}
		if _, err := f.PutPrivate(reader, catalog.EntryPrivate{Slug: "entry-one", Doc: `{}`}); !errors.Is(err, catalog.ErrEntryNotFound) {
			t.Fatalf("standing %d write private: want ErrEntryNotFound, got %v", who, err)
		}
		e, err := f.OpenEntry(reader, "entry-one", "")
		if err != nil {
			t.Fatalf("standing %d open: %v", who, err)
		}
		if strings.Contains(e.Doc, "margin") || strings.Contains(e.Doc, "cost") {
			t.Fatalf("standing %d open carried private parts: %s", who, e.Doc)
		}
	}

	p, err := f.GetPrivate(staff, "entry-one")
	if err != nil || !strings.Contains(p.Doc, "margin") {
		t.Fatalf("staff read private: got %+v, err %v", p, err)
	}
}

func TestAMembersOnlyMountAnswersAnAnonymousRequestNotFound(t *testing.T) {
	cm, f, ctx := fenced(t, catalog.AudienceMembers)
	seed(t, cm, ctx, "entry-one", catalog.VisibilityPublic)

	mux := http.NewServeMux()
	for _, r := range routes.PublicRoutes(f) {
		mux.Handle(r.Method+" "+r.Path, r.Handler)
	}
	open := func(req *http.Request) int {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := open(httptest.NewRequest(http.MethodGet, "/entries/entry-one", nil)); code != http.StatusNotFound {
		t.Fatalf("anonymous request: want 404, got %d", code)
	}
	member := httptest.NewRequest(http.MethodGet, "/entries/entry-one", nil).WithContext(as(ctx, catalog.StandingMember))
	if code := open(member); code != http.StatusOK {
		t.Fatalf("member request: want 200, got %d", code)
	}
}

func TestBelowStaffAPendingEntryAnswersExactlyAsAMissingSlug(t *testing.T) {
	cm, f, ctx := fenced(t, catalog.AudienceMembers)
	seed(t, cm, ctx, "published-one", catalog.VisibilityPublic)
	seed(t, cm, ctx, "pending-one", catalog.VisibilityPending)
	member := as(ctx, catalog.StandingMember)

	_, missingGet := f.GetEntry(member, "no-such-entry")
	_, pendingGet := f.GetEntry(member, "pending-one")
	if !errors.Is(pendingGet, catalog.ErrEntryNotFound) || pendingGet.Error() != missingGet.Error() {
		t.Fatalf("member get of pending: want %v, got %v", missingGet, pendingGet)
	}
	_, missingOpen := f.OpenEntry(member, "no-such-entry", "")
	_, pendingOpen := f.OpenEntry(member, "pending-one", "")
	if !errors.Is(pendingOpen, catalog.ErrEntryNotFound) || pendingOpen.Error() != missingOpen.Error() {
		t.Fatalf("member open of pending: want %v, got %v", missingOpen, pendingOpen)
	}
	if _, err := f.GetEntry(member, "published-one"); err != nil {
		t.Fatalf("member get of published: %v", err)
	}

	list, err := f.ListEntries(member, catalog.ListFilter{})
	if err != nil || len(list) != 1 || list[0].Slug != "published-one" {
		t.Fatalf("member list: want only published-one, got %+v, err %v", list, err)
	}
	list, err = f.ListEntries(member, catalog.ListFilter{Visibility: catalog.VisibilityPending})
	if err != nil || len(list) != 0 {
		t.Fatalf("member list of pending: want nothing, got %+v, err %v", list, err)
	}
}

func TestBelowStaffAListingLimitCountsPublishedEntriesOnly(t *testing.T) {
	cm, f, ctx := fenced(t, catalog.AudienceMembers)
	seed(t, cm, ctx, "a-pending", catalog.VisibilityPending)
	seed(t, cm, ctx, "b-published", catalog.VisibilityPublic)
	list, err := f.ListEntries(as(ctx, catalog.StandingMember), catalog.ListFilter{Limit: 1})
	if err != nil || len(list) != 1 || list[0].Slug != "b-published" {
		t.Fatalf("member list with limit 1: want b-published, got %+v, err %v", list, err)
	}
}

func TestStaffStillReadEveryEntry(t *testing.T) {
	cm, f, ctx := fenced(t, catalog.AudienceMembers)
	seed(t, cm, ctx, "published-one", catalog.VisibilityPublic)
	seed(t, cm, ctx, "pending-one", catalog.VisibilityPending)
	staff := as(ctx, catalog.StandingStaff)

	if _, err := f.GetEntry(staff, "pending-one"); err != nil {
		t.Fatalf("staff get of pending: %v", err)
	}
	list, err := f.ListEntries(staff, catalog.ListFilter{})
	if err != nil || len(list) != 2 {
		t.Fatalf("staff list: want 2 entries, got %d, err %v", len(list), err)
	}
}

func TestASignedOutReaderOfAMembersCatalogueStillGetsNotFound(t *testing.T) {
	cm, f, ctx := fenced(t, catalog.AudienceMembers)
	seed(t, cm, ctx, "published-one", catalog.VisibilityPublic)
	if _, err := f.GetEntry(ctx, "published-one"); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("signed-out get: want ErrEntryNotFound, got %v", err)
	}
	list, err := f.ListEntries(ctx, catalog.ListFilter{})
	if err != nil || len(list) != 0 {
		t.Fatalf("signed-out list: want nothing, got %d, err %v", len(list), err)
	}
}
