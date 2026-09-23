package mwanachamacatalog_test

import (
	"errors"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

// The storefront read must not let a caller tell an unpublished agency from
// one that does not exist. Every refusal is the same error, so the catalog
// cannot be enumerated by probing slugs.
func TestOpenEntry_RefusalsAreIndistinguishable(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "pending-one", catalog.VisibilityPending)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	for _, tc := range []struct{ name, slug, key string }{
		{"missing slug", "no-such-agency", ""},
		{"missing slug with a key", "no-such-agency", "abc"},
		{"pending", "pending-one", ""},
		{"pending with a key", "pending-one", "abc"},
		{"private without a key", "private-one", ""},
		{"private with the wrong key", "private-one", "abc"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := cm.OpenEntry(ctx, tc.slug, tc.key); !errors.Is(err, catalog.ErrEntryNotFound) {
				t.Fatalf("err = %v, want ErrEntryNotFound", err)
			}
		})
	}
}

func TestOpenEntry_PublicOpensWithoutAKey(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "public-one", catalog.VisibilityPublic)

	got, err := cm.OpenEntry(ctx, "public-one", "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if got.Doc != doc {
		t.Errorf("Doc = %q, want the stored document", got.Doc)
	}
}

func TestOpenEntry_PrivateOpensWithItsKey(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	_, raw, err := cm.IssueShareLink(ctx, "private-one", "for the pilot", "owner")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	got, err := cm.OpenEntry(ctx, "private-one", raw)
	if err != nil {
		t.Fatalf("open with key: %v", err)
	}
	if got.Doc != doc {
		t.Errorf("Doc = %q, want the stored document", got.Doc)
	}
}

// A key issued for one agency must not open another.
func TestOpenEntry_KeyIsScopedToItsEntry(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)
	seed(t, cm, ctx, "private-two", catalog.VisibilityPrivate)

	_, raw, err := cm.IssueShareLink(ctx, "private-one", "", "owner")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := cm.OpenEntry(ctx, "private-two", raw); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("err = %v, want ErrEntryNotFound", err)
	}
}

// Re-sending a changed document un-approves it, so nothing unreviewed is
// ever readable. Re-sending the same document must not, or a mining run
// would un-publish every agency it re-visits.
func TestUpsertEntry_ChangedDocReturnsToPending(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "agency-one", catalog.VisibilityPublic)

	same, err := cm.UpsertEntry(ctx, catalog.Entry{
		Slug: "agency-one", Name: "Agency agency-one", Summary: "a summary", Doc: doc,
	})
	if err != nil {
		t.Fatalf("upsert unchanged: %v", err)
	}
	if same.Visibility != catalog.VisibilityPublic {
		t.Errorf("unchanged doc: Visibility = %q, want public", same.Visibility)
	}

	changed, err := cm.UpsertEntry(ctx, catalog.Entry{
		Slug: "agency-one", Name: "Agency agency-one", Summary: "a summary",
		Doc: `{"agency":{"name":"Changed"}}`,
	})
	if err != nil {
		t.Fatalf("upsert changed: %v", err)
	}
	if changed.Visibility != catalog.VisibilityPending {
		t.Errorf("changed doc: Visibility = %q, want pending", changed.Visibility)
	}
	if changed.ApprovedAt != "" {
		t.Errorf("changed doc: ApprovedAt = %q, want cleared", changed.ApprovedAt)
	}
}

// Leaving private must stop the keys working. A link left active would open
// the agency again the moment it was made private a second time.
func TestSetVisibility_AwayFromPrivateRetiresKeys(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	_, raw, err := cm.IssueShareLink(ctx, "private-one", "", "owner")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := cm.SetVisibility(ctx, "private-one", catalog.VisibilityPending); err != nil {
		t.Fatalf("to pending: %v", err)
	}
	if _, err := cm.SetVisibility(ctx, "private-one", catalog.VisibilityPrivate); err != nil {
		t.Fatalf("back to private: %v", err)
	}

	if _, err := cm.OpenEntry(ctx, "private-one", raw); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("err = %v, want the old key to be dead", err)
	}

	links, err := cm.ListShareLinks(ctx, "private-one")
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("len(links) = %d, want the retired link kept", len(links))
	}
	if links[0].Status != catalog.LinkStatusRetired || links[0].RetiredAt == "" {
		t.Errorf("link = %+v, want retired with a timestamp", links[0])
	}
}

// A listing is the card. Carrying Doc would put 34 KB per row into every
// storefront index page.
func TestListEntries_NeverCarriesTheDocument(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "public-one", catalog.VisibilityPublic)

	got, err := cm.ListEntries(ctx, catalog.ListFilter{Visibility: catalog.VisibilityPublic})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Doc != "" {
		t.Errorf("Doc = %q, want empty in a listing", got[0].Doc)
	}
	if got[0].Name == "" || got[0].Slug == "" {
		t.Errorf("card = %+v, want the card fields populated", got[0])
	}
}

func TestListEntries_FiltersAndSearch(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "public-one", catalog.VisibilityPublic)
	seed(t, cm, ctx, "public-two", catalog.VisibilityPublic)
	seed(t, cm, ctx, "pending-one", catalog.VisibilityPending)

	pub, err := cm.ListEntries(ctx, catalog.ListFilter{Visibility: catalog.VisibilityPublic})
	if err != nil {
		t.Fatalf("list public: %v", err)
	}
	if len(pub) != 2 {
		t.Errorf("public count = %d, want 2", len(pub))
	}

	hit, err := cm.ListEntries(ctx, catalog.ListFilter{Search: "PUBLIC-TWO"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hit) != 1 || hit[0].Slug != "public-two" {
		t.Errorf("search = %+v, want just public-two", hit)
	}

	// The domain's own field is filtered by path, not by a column this
	// package knows the name of.
	hit2, err := cm.ListEntries(ctx, catalog.ListFilter{
		Docs: []catalog.DocFilter{{Path: "_template.sector", Value: "health"}},
	})
	if err != nil {
		t.Fatalf("doc filter: %v", err)
	}
	if len(hit2) != 3 {
		t.Errorf("doc filter matched %d, want all 3 seeded health entries", len(hit2))
	}

	none, err := cm.ListEntries(ctx, catalog.ListFilter{
		Docs: []catalog.DocFilter{{Path: "_template.sector", Value: "mining"}},
	})
	if err != nil {
		t.Fatalf("doc filter miss: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("doc filter matched %d, want 0", len(none))
	}
}

func TestCreateEntry_AlwaysLandsPending(t *testing.T) {
	cm, ctx := newManager(t)

	got, err := cm.CreateEntry(ctx, catalog.Entry{
		Slug: "agency-one", Name: "One", Doc: doc, Visibility: catalog.VisibilityPublic,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.Visibility != catalog.VisibilityPending {
		t.Errorf("Visibility = %q, want pending however it was asked for", got.Visibility)
	}
}

func TestCreateEntry_RejectsBadInput(t *testing.T) {
	cm, ctx := newManager(t)

	for _, tc := range []struct {
		name string
		e    catalog.Entry
	}{
		{"empty slug", catalog.Entry{Name: "One", Doc: doc}},
		{"slug with spaces", catalog.Entry{Slug: "not a slug", Name: "One", Doc: doc}},
		{"slug in caps", catalog.Entry{Slug: "NotASlug", Name: "One", Doc: doc}},
		{"no name", catalog.Entry{Slug: "agency-one", Doc: doc}},
		{"doc is not json", catalog.Entry{Slug: "agency-one", Name: "One", Doc: "not json"}},
		{"doc is an array", catalog.Entry{Slug: "agency-one", Name: "One", Doc: `[1,2]`}},
		{"doc is empty", catalog.Entry{Slug: "agency-one", Name: "One"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := cm.CreateEntry(ctx, tc.e); !errors.Is(err, catalog.ErrInvalidEntry) {
				t.Fatalf("err = %v, want ErrInvalidEntry", err)
			}
		})
	}
}

func TestCreateEntry_RejectsADuplicateSlug(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "agency-one", catalog.VisibilityPending)

	_, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "agency-one", Name: "Another", Doc: doc})
	if !errors.Is(err, catalog.ErrEntryExists) {
		t.Fatalf("err = %v, want ErrEntryExists", err)
	}
}

func TestDeleteEntry_HidesItFromEveryRead(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "public-one", catalog.VisibilityPublic)

	if err := cm.DeleteEntry(ctx, "public-one"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := cm.OpenEntry(ctx, "public-one", ""); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Errorf("open: err = %v, want ErrEntryNotFound", err)
	}
	if _, err := cm.GetEntry(ctx, "public-one"); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Errorf("get: err = %v, want ErrEntryNotFound", err)
	}
	got, err := cm.ListEntries(ctx, catalog.ListFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("list = %+v, want empty", got)
	}
}

// The document survives a round trip through the JSONDoc column type
// unchanged. On Postgres that column is jsonb; here it is text.
func TestEntryDoc_RoundTripsUnchanged(t *testing.T) {
	cm, ctx := newManager(t)
	big := `{"agency":{"name":"Big"},"roles":[` +
		strings.Repeat(`{"code":"R-1","description":"a role"},`, 500) +
		`{"code":"R-last"}]}`

	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "big-one", Name: "Big", Doc: big}); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := cm.GetEntry(ctx, "big-one")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Doc != big {
		t.Errorf("document changed in the store (len %d, want %d)", len(got.Doc), len(big))
	}
}

// A document path reaches the database as SQL text, not as a bound
// parameter — neither Postgres's `#>> '{a,b}'` nor SQLite's
// `json_extract(doc,'$.a.b')` takes a placeholder there. Anything that could
// close the quote and continue the statement must be refused outright.
func TestListEntries_RefusesAPathThatIsNotAPlainPath(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "public-one", catalog.VisibilityPublic)

	for _, path := range []string{
		"_template.sector'}' = '' OR '1'='1",
		"a'; drop table test_entries; --",
		"a,b",
		"$.a",
		"a b",
		"a['b']",
		"",
		".",
		"a..b",
	} {
		t.Run(path, func(t *testing.T) {
			_, err := cm.ListEntries(ctx, catalog.ListFilter{
				Docs: []catalog.DocFilter{{Path: path, Value: "x"}},
			})
			if !errors.Is(err, catalog.ErrInvalidDocPath) {
				t.Fatalf("err = %v, want ErrInvalidDocPath", err)
			}
		})
	}

	// The table is still there and still readable.
	got, err := cm.ListEntries(ctx, catalog.ListFilter{})
	if err != nil {
		t.Fatalf("list after the refused paths: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want the seeded entry intact", len(got))
	}
}
