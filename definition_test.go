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

func newDefinitionMount(t *testing.T) (*gorm.DB, *spec.Spec) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	s, err := catalog.LoadSpec(filepath.Join(".", "spec", "examples", "agency.catalog.json"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db, s
}

func TestPutDefinition_KindsWrittenByTheHostAreCheckedOnWrite(t *testing.T) {
	db, s := newDefinitionMount(t)
	if err := catalog.PutDefinition(db, s, map[string]any{"kinds": placeKinds}); err != nil {
		t.Fatalf("put definition: %v", err)
	}
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	ctx := context.Background()
	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "hall", Name: "Hall", Doc: `{"kind":"place","seats":"many"}`}); !errors.Is(err, catalog.ErrEntryKind) {
		t.Fatalf("an entry missing its kind's required field: want ErrEntryKind, got %v", err)
	}
	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "hall", Name: "Hall", Doc: `{"kind":"place","address":"Main St"}`}); err != nil {
		t.Fatalf("an entry fitting its kind: %v", err)
	}
}

func TestPutDefinition_MalformedKindsAreRefusedAndNothingIsWritten(t *testing.T) {
	db, s := newDefinitionMount(t)
	malformed := []any{map[string]any{"name": "place", "fields": []any{map[string]any{"name": "kind", "type": "string"}}}}
	if err := catalog.PutDefinition(db, s, map[string]any{"kinds": malformed}); !errors.Is(err, models.ErrKindsMalformed) {
		t.Fatalf("want ErrKindsMalformed, got %v", err)
	}
	definition, _, err := spec.LoadSettingLayers(db, s)
	if err != nil {
		t.Fatalf("load layers: %v", err)
	}
	if len(definition) != 0 {
		t.Fatalf("a refused definition wrote %v", definition)
	}
}
