package mwanachamacatalog_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

// A document carrying a domain-shaped field, to prove the store filters on
// it by path without knowing the word "sector".
const doc = `{"agency":{"name":"Test"},"_template":{"sector":"health"},"modules":[{"code":"auth","enabled":true}]}`

func newManager(t *testing.T) (catalog.CatalogManager, context.Context) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	tables := catalog.DefaultTableNames("test")
	if err := catalog.Migrate(db, tables); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cm, err := catalog.NewCatalogManager(db, tables)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	return cm, context.Background()
}

// seed stores one entry and moves it to v, returning it.
func seed(t *testing.T, cm catalog.CatalogManager, ctx context.Context, slug string, v catalog.Visibility) catalog.Entry {
	t.Helper()
	e, err := cm.CreateEntry(ctx, catalog.Entry{
		Slug: slug, Name: "Agency " + slug, Summary: "a summary", Doc: doc,
	})
	if err != nil {
		t.Fatalf("create %s: %v", slug, err)
	}
	if v == catalog.VisibilityPending {
		return e
	}
	e, err = cm.SetVisibility(ctx, slug, v)
	if err != nil {
		t.Fatalf("set visibility %s: %v", slug, err)
	}
	return e
}
