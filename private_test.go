package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

const configs = `{"actor":{"module_code":"actor"},"comm":{"module_code":"comm"}}`

// An empty field means "leave it alone", so re-ranking the catalog does not
// erase every readme that is not resent.
func TestPutPrivate_PartialWriteKeepsWhatItOmits(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "agency-one", catalog.VisibilityPending)

	if _, err := cm.PutPrivate(ctx, catalog.EntryPrivate{
		Slug: "agency-one", Readme: "the readme", ModuleConfigs: configs, Rank: 3,
	}); err != nil {
		t.Fatalf("first put: %v", err)
	}

	got, err := cm.PutPrivate(ctx, catalog.EntryPrivate{Slug: "agency-one", Rank: 1})
	if err != nil {
		t.Fatalf("second put: %v", err)
	}
	if got.Readme != "the readme" {
		t.Errorf("Readme = %q, want it kept", got.Readme)
	}
	if got.ModuleConfigs != configs {
		t.Errorf("ModuleConfigs = %q, want it kept", got.ModuleConfigs)
	}
	if got.Rank != 1 {
		t.Errorf("Rank = %d, want 1", got.Rank)
	}
}

// The private parts are a separate read, so nothing here can leak through
// the storefront's view of the entry.
func TestPutPrivate_NeverReachesTheDocument(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "public-one", catalog.VisibilityPublic)

	if _, err := cm.PutPrivate(ctx, catalog.EntryPrivate{
		Slug: "public-one", Readme: "internal only", Sales: "a prospect list", ModuleConfigs: configs,
	}); err != nil {
		t.Fatalf("put: %v", err)
	}

	opened, err := cm.OpenEntry(ctx, "public-one", "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if opened.Doc != doc {
		t.Errorf("Doc = %q, want the stored document untouched", opened.Doc)
	}

	listed, err := cm.ListEntries(ctx, catalog.ListFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 1 || listed[0].Doc != "" {
		t.Errorf("listing = %+v, want a card with no document", listed)
	}
}

func TestGetPrivate_EmptyForAnEntryWithNothingWritten(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "agency-one", catalog.VisibilityPending)

	got, err := cm.GetPrivate(ctx, "agency-one")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Slug != "agency-one" || got.Readme != "" || got.Rank != 0 {
		t.Errorf("got = %+v, want empty private parts", got)
	}
}

func TestPutPrivate_RejectsBadInput(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "agency-one", catalog.VisibilityPending)

	for _, tc := range []struct {
		name string
		p    catalog.EntryPrivate
	}{
		{"negative rank", catalog.EntryPrivate{Slug: "agency-one", Rank: -1}},
		{"configs are not json", catalog.EntryPrivate{Slug: "agency-one", ModuleConfigs: "not json"}},
		{"configs are an array", catalog.EntryPrivate{Slug: "agency-one", ModuleConfigs: `[1,2]`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := cm.PutPrivate(ctx, tc.p); !errors.Is(err, catalog.ErrInvalidPrivate) {
				t.Fatalf("err = %v, want ErrInvalidPrivate", err)
			}
		})
	}
}

func TestPutPrivate_UnknownEntry(t *testing.T) {
	cm, ctx := newManager(t)
	if _, err := cm.PutPrivate(ctx, catalog.EntryPrivate{Slug: "no-such-agency", Readme: "x"}); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("err = %v, want ErrEntryNotFound", err)
	}
}
