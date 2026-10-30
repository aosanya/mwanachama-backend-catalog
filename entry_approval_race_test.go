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

func TestCAT35_UnchangedResendKeepsAReviewersApproval(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:cat35?mode=memory&cache=shared"), &gorm.Config{
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

	pending := seed(t, cm, ctx, "beta", models.VisibilityPending)
	entries, ok := s.ByRole("entry")
	if !ok {
		t.Fatal("the shipped spec fills no entry role")
	}
	entryTable := s.TableFor(entries)

	var approved bool
	fired := false
	err = db.Callback().Update().Before("gorm:update").Register("cat35_approve_mid_resend", func(tx *gorm.DB) {
		row, isRow := tx.Statement.Dest.(map[string]any)
		if fired || tx.Statement.Table != entryTable || !isRow {
			return
		}
		if _, writesDoc := row["doc"]; !writesDoc {
			return
		}
		fired = true
		if _, err := cm.SetVisibility(context.Background(), "beta", models.VisibilityPublic); err == nil {
			approved = true
		}
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}

	_, resendErr := cm.UpsertEntry(ctx, catalog.Entry{Slug: "beta", Name: pending.Name, Summary: pending.Summary, Doc: pending.Doc})
	if !fired {
		t.Fatalf("precondition: UpsertEntry never reached its write (err %v)", resendErr)
	}
	if !approved {
		t.Fatalf("precondition: the reviewer's SetVisibility(public) did not succeed")
	}
	if resendErr != nil {
		return
	}

	stored, err := cm.GetEntry(ctx, "beta")
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	_, openErr := cm.OpenEntry(ctx, "beta", "")
	if stored.Visibility != models.VisibilityPublic || stored.ApprovedAt == "" || !errors.Is(openErr, nil) {
		t.Errorf("CAT35 open bug: a reviewer approved beta (visibility public, approved_at set) while an unchanged re-send of its pending document was in flight, both calls succeeded, and the entry now reads visibility=%q approved_at=%q and OpenEntry (anonymous, no key) answers err=%v; want visibility=%q, a non-empty approved_at, and OpenEntry to succeed — re-sending an unchanged document must leave the review alone rather than write back the pre-approval state it read before the approval committed", stored.Visibility, stored.ApprovedAt, openErr, models.VisibilityPublic)
	}
}
