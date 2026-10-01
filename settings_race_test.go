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

func TestCAT23_OpenBug_ConcurrentPutSettingOnANewKeyFailsInstead(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:cat23?mode=memory&cache=shared"), &gorm.Config{
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

	settingRole, ok := s.ByRole("setting")
	if !ok {
		t.Fatal("the shipped spec fills no setting role")
	}
	settingTable := s.TableFor(settingRole)

	fired := false
	var raceErr error
	err = db.Callback().Create().Before("gorm:create").Register("cat23_concurrent_put_setting", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != settingTable {
			return
		}
		fired = true
		_, raceErr = cm.PutSetting(context.Background(), "appeal_rule", "value-B")
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}

	_, outerErr := cm.PutSetting(ctx, "appeal_rule", "value-A")
	if !fired {
		t.Fatalf("precondition: PutSetting never reached its insert")
	}
	if raceErr != nil {
		t.Fatalf("precondition: the concurrent PutSetting that won the race failed: %v", raceErr)
	}
	if outerErr != nil {
		t.Errorf("CAT23 open bug: PutSetting(\"appeal_rule\", \"value-A\") failed with %v while a concurrent PutSetting for the same new key was committing; want it to succeed, because two legitimate writers setting a knob for the first time at once should both see their write land, not surface a raw SQL constraint failure to one of them", outerErr)
	}

	got, err := cm.GetSetting(ctx, "appeal_rule")
	if err != nil {
		t.Fatalf("get setting: %v", err)
	}
	if got.Value != "value-A" && got.Value != "value-B" {
		t.Errorf("CAT23 open bug: after the race, GetSetting(\"appeal_rule\") = %q, want either racing writer's value", got.Value)
	}
}
