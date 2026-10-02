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

func TestCAT33_OpenBug_ConcurrentPutPrivateOnDifferentFieldsLosesOne(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:cat33?mode=memory&cache=shared"), &gorm.Config{
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
		Slug: "cat33-slug", Name: "CAT33", Summary: "a summary",
		Doc: `{"agency":{"name":"CAT33"},"_template":{"sector":"health"},"modules":[]}`,
	}); err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if _, err := cm.PutPrivate(ctx, catalog.EntryPrivate{Slug: "cat33-slug", Readme: "base-readme"}); err != nil {
		t.Fatalf("base put private: %v", err)
	}

	privRole, ok := s.ByRole("entry_private")
	if !ok {
		t.Fatal("the shipped spec fills no entry_private role")
	}
	privTable := s.TableFor(privRole)

	fired := false
	var raceErr error
	err = db.Callback().Update().Before("gorm:update").Register("cat33_concurrent_putprivate", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != privTable {
			return
		}
		fired = true
		_, raceErr = cm.PutPrivate(context.Background(), catalog.EntryPrivate{Slug: "cat33-slug", Sales: "concurrent-sales"})
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}

	if _, err := cm.PutPrivate(ctx, catalog.EntryPrivate{Slug: "cat33-slug", Readme: "writer-A-readme"}); err != nil {
		t.Fatalf("outer put private: %v", err)
	}
	if !fired {
		t.Fatalf("precondition: the outer PutPrivate never reached its update")
	}
	if raceErr != nil {
		t.Fatalf("precondition: the concurrent PutPrivate that ran mid-update failed: %v", raceErr)
	}

	final, err := cm.GetPrivate(ctx, "cat33-slug")
	if err != nil {
		t.Fatalf("get private: %v", err)
	}
	if final.Readme != "writer-A-readme" {
		t.Errorf("CAT33 open bug: after two concurrent PutPrivate calls setting different fields, Readme = %q, want %q (writer A's own field)", final.Readme, "writer-A-readme")
	}
	if final.Sales != "concurrent-sales" {
		t.Errorf("CAT33 open bug: after two concurrent PutPrivate calls setting different fields (writer A set Readme, writer B set Sales), Sales = %q, want %q — writer A's read-merge-write clobbered writer B's field with the stale value it read before B's write committed, even though PutPrivate's own partial-write contract (private.go) says an empty field means \"leave it alone\" precisely so one caller's write does not erase another's", final.Sales, "concurrent-sales")
	}
}
