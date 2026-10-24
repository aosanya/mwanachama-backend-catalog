package mwanachamacatalog_test

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func newSeededManager(t *testing.T, seeds any, override any) (catalog.CatalogManager, context.Context) {
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
	if seeds != nil {
		if err := spec.PutDefinitionSettings(db, s, map[string]any{"seeds": seeds}); err != nil {
			t.Fatalf("definition: %v", err)
		}
	}
	if override != nil {
		if _, err := spec.StoreOverride(db, s, "seeds", override, "steward", "2026-10-24T00:00:00Z"); err != nil {
			t.Fatalf("override: %v", err)
		}
	}
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	return cm, context.Background()
}

var twoSeeds = []any{
	map[string]any{"slug": "first", "name": "First", "doc": map[string]any{"note": "a"}},
	map[string]any{"slug": "second", "name": "Second", "summary": "the second"},
}

func TestSeedEntries_NoSeedsDeclaredCreatesNothing(t *testing.T) {
	cm, ctx := newSeededManager(t, nil, nil)
	created, err := cm.SeedEntries(ctx)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if len(created) != 0 {
		t.Fatalf("created %v with no seeds declared", created)
	}
}

func TestSeedEntries_AMountStartsWithItsSeedsAsPending(t *testing.T) {
	cm, ctx := newSeededManager(t, twoSeeds, nil)
	created, err := cm.SeedEntries(ctx)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if !reflect.DeepEqual(created, []string{"first", "second"}) {
		t.Fatalf("created %v", created)
	}
	e, err := cm.GetEntry(ctx, "first")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if e.Name != "First" || e.Visibility != models.VisibilityPending || e.Doc != `{"note":"a"}` {
		t.Fatalf("seeded entry %+v", e)
	}
	e, err = cm.GetEntry(ctx, "second")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if e.Summary != "the second" || e.Doc != "{}" {
		t.Fatalf("seeded entry %+v", e)
	}
}

func TestSeedEntries_ApplyingTwiceCreatesNothingTheSecondTime(t *testing.T) {
	cm, ctx := newSeededManager(t, twoSeeds, nil)
	if _, err := cm.SeedEntries(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}
	created, err := cm.SeedEntries(ctx)
	if err != nil {
		t.Fatalf("second seed: %v", err)
	}
	if len(created) != 0 {
		t.Fatalf("second seed created %v", created)
	}
}

func TestSeedEntries_AnExistingEntryIsNotOverwritten(t *testing.T) {
	cm, ctx := newSeededManager(t, twoSeeds, nil)
	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "first", Name: "Mine", Doc: `{"note":"mine"}`}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := cm.SetVisibility(ctx, "first", models.VisibilityPublic); err != nil {
		t.Fatalf("publish: %v", err)
	}
	created, err := cm.SeedEntries(ctx)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if !reflect.DeepEqual(created, []string{"second"}) {
		t.Fatalf("created %v", created)
	}
	e, err := cm.GetEntry(ctx, "first")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if e.Name != "Mine" || e.Doc != `{"note":"mine"}` || e.Visibility != models.VisibilityPublic {
		t.Fatalf("existing entry overwritten: %+v", e)
	}
}

func TestSeedEntries_ADeletedSeedIsNotRecreated(t *testing.T) {
	cm, ctx := newSeededManager(t, twoSeeds, nil)
	if _, err := cm.SeedEntries(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := cm.DeleteEntry(ctx, "first"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	created, err := cm.SeedEntries(ctx)
	if err != nil {
		t.Fatalf("seed again: %v", err)
	}
	if len(created) != 0 {
		t.Fatalf("seed again created %v", created)
	}
	if _, err := cm.GetEntry(ctx, "first"); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("deleted seed came back: %v", err)
	}
}

func TestSeedEntries_AnOverrideReplacesTheDefinitionsSeeds(t *testing.T) {
	cm, ctx := newSeededManager(t, twoSeeds, []any{map[string]any{"slug": "third", "name": "Third"}})
	created, err := cm.SeedEntries(ctx)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if !reflect.DeepEqual(created, []string{"third"}) {
		t.Fatalf("created %v", created)
	}
}

func TestSeedEntries_ASeedIsCheckedAgainstItsKind(t *testing.T) {
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
	if err := spec.PutDefinitionSettings(db, s, map[string]any{
		"kinds": placeKinds,
		"seeds": []any{
			map[string]any{"slug": "fits", "name": "Fits", "doc": map[string]any{"kind": "place", "address": "1 Lane"}},
			map[string]any{"slug": "misfit", "name": "Misfit", "doc": map[string]any{"kind": "place"}},
		},
	}); err != nil {
		t.Fatalf("definition: %v", err)
	}
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	created, err := cm.SeedEntries(context.Background())
	if !errors.Is(err, catalog.ErrEntryKind) {
		t.Fatalf("got %v, want ErrEntryKind", err)
	}
	if !reflect.DeepEqual(created, []string{"fits"}) {
		t.Fatalf("created %v before the misfit", created)
	}
}

func TestSeedEntries_AMalformedSettingIsRefused(t *testing.T) {
	for _, seeds := range []any{
		[]any{map[string]any{"name": "No slug"}},
		[]any{map[string]any{"slug": "x"}},
		[]any{map[string]any{"slug": "x", "name": "X"}, map[string]any{"slug": "x", "name": "Again"}},
		[]any{map[string]any{"slug": "x", "name": "X", "doc": "text"}},
		[]any{map[string]any{"slug": "x", "name": "X", "visibility": "public"}},
	} {
		cm, ctx := newSeededManager(t, seeds, nil)
		if _, err := cm.SeedEntries(ctx); !errors.Is(err, models.ErrSeedsMalformed) {
			t.Fatalf("seeds %v: got %v, want ErrSeedsMalformed", seeds, err)
		}
	}
}
