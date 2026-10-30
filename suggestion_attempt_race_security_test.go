package mwanachamacatalog_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func TestCAT19_FailedAttemptReportDoesNotReviveARejectedSuggestion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:cat19?mode=memory&cache=shared"), &gorm.Config{
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

	sg := file(t, cm, ctx, "clinic")
	if _, err := cm.DecideSuggestion(ctx, sg.ID, models.SuggestionStatusApproved, nil); err != nil {
		t.Fatalf("precondition: approve: %v", err)
	}
	suggestions, ok := s.ByRole("suggestion")
	if !ok {
		t.Fatal("the shipped spec fills no suggestion role")
	}
	suggestionTable := s.TableFor(suggestions)

	var rejected bool
	fired := false
	err = db.Callback().Update().Before("gorm:update").Register("cat19_reject_mid_attempt", func(tx *gorm.DB) {
		row, isRow := tx.Statement.Dest.(map[string]any)
		if fired || tx.Statement.Table != suggestionTable || !isRow {
			return
		}
		if _, countsAttempt := row["attempts"]; !countsAttempt {
			return
		}
		if _, decides := row["decided_at"]; decides {
			return
		}
		fired = true
		if _, err := cm.DecideSuggestion(context.Background(), sg.ID, models.SuggestionStatusRejected, nil); err == nil {
			rejected = true
		}
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}

	_, attemptErr := cm.RecordSuggestionAttempt(ctx, sg.ID, "run-1", "the build failed")
	if !fired {
		t.Fatalf("precondition: RecordSuggestionAttempt never reached its write (err %v)", attemptErr)
	}
	if !rejected {
		t.Fatalf("precondition: the reviewer's DecideSuggestion(rejected) did not succeed")
	}

	got, err := cm.GetSuggestion(ctx, sg.ID)
	if err != nil {
		t.Fatalf("GetSuggestion: %v", err)
	}
	if got.Status != models.SuggestionStatusRejected {
		t.Errorf("CAT19 (P2): a reviewer rejected suggestion clinic while a failed-attempt report was in flight, the reject succeeded and the report answered %v, and the suggestion now reads %q with %d attempts; want it to stay rejected, because an attempt may only count against a suggestion that is still approved at the moment it is written", attemptErr, got.Status, got.Attempts)
	}
}
