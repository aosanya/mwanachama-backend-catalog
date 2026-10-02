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

func TestCAT34_OpenBug_ConcurrentPutPrivateOnANewSlugFailsInstead(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:cat34?mode=memory&cache=shared"), &gorm.Config{
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
		Slug: "cat34-slug", Name: "CAT34", Summary: "a summary",
		Doc: `{"agency":{"name":"CAT34"},"_template":{"sector":"health"},"modules":[]}`,
	}); err != nil {
		t.Fatalf("create entry: %v", err)
	}

	privRole, ok := s.ByRole("entry_private")
	if !ok {
		t.Fatal("the shipped spec fills no entry_private role")
	}
	privTable := s.TableFor(privRole)

	fired := false
	var raceErr error
	err = db.Callback().Create().Before("gorm:create").Register("cat34_concurrent_putprivate_insert", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != privTable {
			return
		}
		fired = true
		_, raceErr = cm.PutPrivate(context.Background(), catalog.EntryPrivate{Slug: "cat34-slug", Sales: "concurrent-sales"})
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}

	_, outerErr := cm.PutPrivate(ctx, catalog.EntryPrivate{Slug: "cat34-slug", Readme: "writer-A-readme"})
	if !fired {
		t.Fatalf("precondition: PutPrivate never reached its insert")
	}
	if raceErr != nil {
		t.Fatalf("precondition: the concurrent PutPrivate that won the race failed: %v", raceErr)
	}
	if outerErr != nil {
		t.Errorf("CAT34 open bug: PutPrivate(\"cat34-slug\", Readme: \"writer-A-readme\") failed with %v while a concurrent first-time PutPrivate for the same new slug was committing; want it to succeed (recovering into the merge path, the same way a concurrent PutSetting on a new key should per CAT23), not surface a raw SQL constraint failure to one of two legitimate writers", outerErr)
	}

	final, err := cm.GetPrivate(ctx, "cat34-slug")
	if err != nil {
		t.Fatalf("get private: %v", err)
	}
	if final.Readme == "" && final.Sales == "" {
		t.Errorf("CAT34 open bug: after the race, GetPrivate(\"cat34-slug\") carries neither writer's field (Readme=%q Sales=%q); want at least one writer's value stored", final.Readme, final.Sales)
	}
}
