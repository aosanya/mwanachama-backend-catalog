package mwanachamacatalog_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func TestCAT18_UnchangedResendKeepsAReviewersTakedown(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:cat18?mode=memory&cache=shared"), &gorm.Config{
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

	published := seed(t, cm, ctx, "alpha", models.VisibilityPublic)
	entries, ok := s.ByRole("entry")
	if !ok {
		t.Fatal("the shipped spec fills no entry role")
	}
	entryTable := s.TableFor(entries)

	var takenDown bool
	fired := false
	err = db.Callback().Update().Before("gorm:update").Register("cat18_takedown_mid_resend", func(tx *gorm.DB) {
		row, isRow := tx.Statement.Dest.(map[string]any)
		if fired || tx.Statement.Table != entryTable || !isRow {
			return
		}
		if _, writesDoc := row["doc"]; !writesDoc {
			return
		}
		fired = true
		if _, err := cm.SetVisibility(context.Background(), "alpha", models.VisibilityPrivate); err == nil {
			takenDown = true
		}
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}

	_, resendErr := cm.UpsertEntry(ctx, catalog.Entry{Slug: "alpha", Name: published.Name, Summary: published.Summary, Doc: published.Doc})
	if !fired {
		t.Fatalf("precondition: UpsertEntry never reached its write (err %v)", resendErr)
	}
	if !takenDown {
		t.Fatalf("precondition: the reviewer's SetVisibility(private) did not succeed")
	}
	if resendErr != nil {
		return
	}

	stored, err := cm.GetEntry(ctx, "alpha")
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	_, openErr := cm.OpenEntry(ctx, "alpha", "")
	if stored.Visibility != models.VisibilityPrivate || !errors.Is(openErr, catalog.ErrEntryNotFound) {
		t.Errorf("CAT18 (P1) open hole: a reviewer moved alpha to private while an unchanged re-send of its document was in flight, both calls succeeded, and the entry reads %q and opens to an anonymous reader with no key (OpenEntry err %v); want it private and ErrEntryNotFound, because re-sending an unchanged document must leave the review alone rather than write back the visibility it read", stored.Visibility, openErr)
	}
}
