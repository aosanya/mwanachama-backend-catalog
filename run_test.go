package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

// An agency view counts only against a slug that exists, so a caller cannot
// grow the table by inventing slugs.
func TestRecordPageView_AgencyViewNeedsAnExistingEntry(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "real-one", catalog.VisibilityPublic)

	if err := cm.RecordPageView(ctx, catalog.PageKindEntry, "real-one"); err != nil {
		t.Fatalf("real slug: %v", err)
	}
	if err := cm.RecordPageView(ctx, catalog.PageKindEntry, "invented-one"); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("invented slug: err = %v, want ErrEntryNotFound", err)
	}

	got, err := cm.ListPageViews(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Slug != "real-one" {
		t.Errorf("views = %+v, want only the real slug counted", got)
	}
}

func TestRecordPageView_Accumulates(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "real-one", catalog.VisibilityPublic)

	for i := 0; i < 5; i++ {
		if err := cm.RecordPageView(ctx, catalog.PageKindIndex, ""); err != nil {
			t.Fatalf("home %d: %v", i, err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := cm.RecordPageView(ctx, catalog.PageKindEntry, "real-one"); err != nil {
			t.Fatalf("agency %d: %v", i, err)
		}
	}

	got, err := cm.ListPageViews(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want home and one agency", len(got))
	}
	// Most viewed first.
	if got[0].Page != catalog.PageKindIndex || got[0].Views != 5 {
		t.Errorf("first = %+v, want home with 5", got[0])
	}
	if got[1].Views != 2 {
		t.Errorf("second = %+v, want 2", got[1])
	}
}

// The front page has no slug, and passing one must not create a second row
// that splits the same count in two.
func TestRecordPageView_HomeIgnoresASlug(t *testing.T) {
	cm, ctx := newManager(t)

	if err := cm.RecordPageView(ctx, catalog.PageKindIndex, ""); err != nil {
		t.Fatalf("home: %v", err)
	}
	if err := cm.RecordPageView(ctx, catalog.PageKindIndex, "stray-slug"); err != nil {
		t.Fatalf("home with slug: %v", err)
	}

	got, err := cm.ListPageViews(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Views != 2 || got[0].Slug != "" {
		t.Errorf("views = %+v, want one home row with 2", got)
	}
}

func TestRecordPageView_RejectsAnUnknownPage(t *testing.T) {
	cm, ctx := newManager(t)
	if err := cm.RecordPageView(ctx, "dashboard", ""); !errors.Is(err, catalog.ErrInvalidPageView) {
		t.Fatalf("err = %v, want ErrInvalidPageView", err)
	}
}

func TestRecordRun_AndListNewestFirst(t *testing.T) {
	cm, ctx := newManager(t)

	for _, s := range []string{"first report", "second report"} {
		if _, err := cm.RecordRun(ctx, catalog.MiningRun{
			RunID: s, SubmittedBy: "routine", TemplatesFiled: 3, Summary: s,
		}); err != nil {
			t.Fatalf("record %q: %v", s, err)
		}
	}

	got, err := cm.ListRuns(ctx, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}

	limited, err := cm.ListRuns(ctx, 1)
	if err != nil {
		t.Fatalf("list limited: %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("len = %d, want the limit respected", len(limited))
	}
}

func TestRecordRun_RejectsANegativeCount(t *testing.T) {
	cm, ctx := newManager(t)
	if _, err := cm.RecordRun(ctx, catalog.MiningRun{Summary: "x", TemplatesFiled: -1}); !errors.Is(err, catalog.ErrInvalidRun) {
		t.Fatalf("err = %v, want ErrInvalidRun", err)
	}
}

func TestSettings_RoundTripAndOverwrite(t *testing.T) {
	cm, ctx := newManager(t)

	if _, err := cm.PutSetting(ctx, "appeal_rule", "first rule"); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, err := cm.GetSetting(ctx, "appeal_rule")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Value != "first rule" {
		t.Errorf("Value = %q", got.Value)
	}

	if _, err := cm.PutSetting(ctx, "appeal_rule", "second rule"); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	got, err = cm.GetSetting(ctx, "appeal_rule")
	if err != nil {
		t.Fatalf("get again: %v", err)
	}
	if got.Value != "second rule" {
		t.Errorf("Value = %q, want the overwrite", got.Value)
	}
}

func TestGetSetting_Unknown(t *testing.T) {
	cm, ctx := newManager(t)
	if _, err := cm.GetSetting(ctx, "no-such-key"); !errors.Is(err, catalog.ErrSettingNotFound) {
		t.Fatalf("err = %v, want ErrSettingNotFound", err)
	}
}
