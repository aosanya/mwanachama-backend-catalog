package mwanachamacatalog_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func TestCAT43_OpenBug_ConcurrentAttachSuggestionOnANewPairFailsInstead(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:cat43?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(2)
	t.Cleanup(func() { sqlDB.Close() })

	s, err := catalog.LoadSpec(filepath.Join(".", "spec", "examples", "agency.catalog.json"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	ctx := context.Background()

	if _, err := cm.CreateEntry(ctx, catalog.Entry{
		Slug: "cat43-entry", Name: "CAT43", Summary: "a summary",
		Doc: `{"agency":{"name":"CAT43"},"_template":{"sector":"health"},"modules":[]}`,
	}); err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if _, err := cm.SetVisibility(ctx, "cat43-entry", catalog.VisibilityPublic); err != nil {
		t.Fatalf("set visibility: %v", err)
	}
	sg, err := cm.FileSuggestion(ctx, catalog.Suggestion{
		Slug: "cat43-proposed", Title: "Build it", Why: "worth having",
		Source: catalog.SuggestionSourceRun,
	})
	if err != nil {
		t.Fatalf("file suggestion: %v", err)
	}

	matchRole, ok := s.ByRole("suggestion_match")
	if !ok {
		t.Fatal("the shipped spec fills no suggestion_match role")
	}
	matchTable := s.TableFor(matchRole)

	fired := false
	var raceErr error
	err = db.Callback().Create().Before("gorm:create").Register("cat43_concurrent_attachsuggestion_insert", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != matchTable {
			return
		}
		fired = true
		_, raceErr = cm.AttachSuggestion(context.Background(), sg.ID, "cat43-entry", "concurrent-note")
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}

	_, outerErr := cm.AttachSuggestion(ctx, sg.ID, "cat43-entry", "outer-note")
	if !fired {
		t.Fatalf("precondition: AttachSuggestion never reached its insert")
	}
	if raceErr != nil {
		t.Fatalf("precondition: the concurrent AttachSuggestion that won the race failed: %v", raceErr)
	}
	if outerErr != nil {
		t.Errorf("CAT43 open bug: AttachSuggestion(%q, \"cat43-entry\", \"outer-note\") failed with %v while a concurrent first-time AttachSuggestion for the same new (suggestion, slug) pair was committing; want it to succeed as a no-op or a clean domain error, not surface a raw SQL constraint failure to one of two legitimate callers attaching the same match at once", sg.ID, outerErr)
	}

	matches, err := cm.ListSuggestionMatches(ctx, sg.ID)
	if err != nil {
		t.Fatalf("list suggestion matches: %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("CAT43 open bug: after the race, ListSuggestionMatches(%q) returned %d rows, want exactly 1 for the (suggestion, slug) pair both callers raced to attach", sg.ID, len(matches))
	}
}
