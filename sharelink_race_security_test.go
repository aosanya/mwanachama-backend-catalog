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

func TestCAT17_OpenHole_KeyIssuedWhileLeavingPrivateSurvivesTheRetirement(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:cat17?mode=memory&cache=shared"), &gorm.Config{
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

	seed(t, cm, ctx, "alpha", models.VisibilityPrivate)

	links, ok := s.ByRole("share_link")
	if !ok {
		t.Fatal("the shipped spec fills no share_link role")
	}
	linkTable := s.TableFor(links)

	var leftPrivate bool
	fired := false
	err = db.Callback().Create().Before("gorm:create").Register("cat17_leave_private_mid_issue", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != linkTable {
			return
		}
		fired = true
		if _, err := cm.SetVisibility(context.Background(), "alpha", models.VisibilityPublic); err == nil {
			leftPrivate = true
		}
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}

	_, raw, issueErr := cm.IssueShareLink(ctx, "alpha", "handed out", "admin")
	if !fired {
		t.Fatalf("precondition: IssueShareLink never reached its insert (err %v)", issueErr)
	}
	if issueErr != nil || !leftPrivate {
		return
	}

	if e, err := cm.GetEntry(ctx, "alpha"); err != nil || e.Visibility != models.VisibilityPublic {
		t.Fatalf("precondition: the entry should have left private, got %+v (%v)", e, err)
	}
	if _, err := cm.SetVisibility(ctx, "alpha", models.VisibilityPrivate); err != nil {
		t.Fatalf("precondition: back to private: %v", err)
	}

	_, openErr := cm.OpenEntry(ctx, "alpha", raw)
	if !errors.Is(openErr, catalog.ErrEntryNotFound) {
		t.Errorf("CAT17 (P2) open hole: a share key issued while the entry was leaving private came back active and opens the entry once it is private again (OpenEntry err %v); want ErrEntryNotFound, because leaving private must retire every key, including one minted in the same moment", openErr)
	}
}
