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
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func newKindedManager(t *testing.T, kinds any, override any) (catalog.CatalogManager, context.Context) {
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
	if kinds != nil {
		if err := spec.PutDefinitionSettings(db, s, map[string]any{"kinds": kinds}); err != nil {
			t.Fatalf("definition: %v", err)
		}
	}
	if override != nil {
		if _, err := spec.StoreOverride(db, s, "kinds", override, "steward", "2026-10-24T00:00:00Z"); err != nil {
			t.Fatalf("override: %v", err)
		}
	}
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	return cm, context.Background()
}

var placeKinds = []any{
	map[string]any{"name": "place", "fields": []any{
		map[string]any{"name": "address", "type": "string", "required": true},
		map[string]any{"name": "seats", "type": "number"},
	}},
}

func TestEntryKind_NoKindsDeclaredLeavesEveryEntryAlone(t *testing.T) {
	cm, ctx := newKindedManager(t, nil, nil)
	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "one", Name: "One", Doc: `{"kind":"place","seats":"many"}`}); err != nil {
		t.Fatalf("create with no kinds declared: %v", err)
	}
}

func TestEntryKind_AnEntryNamingNoKindIsNotChecked(t *testing.T) {
	cm, ctx := newKindedManager(t, placeKinds, nil)
	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "one", Name: "One", Doc: `{"seats":"many"}`}); err != nil {
		t.Fatalf("create naming no kind: %v", err)
	}
}

func TestEntryKind_FieldsAreCheckedAgainstTheKindOnEverySave(t *testing.T) {
	cm, ctx := newKindedManager(t, placeKinds, nil)
	for _, tc := range []struct {
		name string
		doc  string
		ok   bool
	}{
		{"fits", `{"kind":"place","address":"1 Lane","seats":4}`, true},
		{"optional absent", `{"kind":"place","address":"1 Lane"}`, true},
		{"required missing", `{"kind":"place","seats":4}`, false},
		{"wrong type", `{"kind":"place","address":"1 Lane","seats":"four"}`, false},
		{"unknown kind", `{"kind":"thing","address":"1 Lane"}`, false},
		{"kind not text", `{"kind":3,"address":"1 Lane"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "c-" + slugOf(tc.name), Name: "C", Doc: tc.doc})
			if tc.ok && err != nil {
				t.Fatalf("create: %v", err)
			}
			if !tc.ok && !errors.Is(err, catalog.ErrEntryKind) {
				t.Fatalf("create: err = %v, want ErrEntryKind", err)
			}
		})
	}

	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "u", Name: "U", Doc: `{"kind":"place","address":"1 Lane"}`}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := cm.UpdateEntry(ctx, catalog.Entry{Slug: "u", Name: "U", Visibility: catalog.VisibilityPending, Doc: `{"kind":"place"}`}); !errors.Is(err, catalog.ErrEntryKind) {
		t.Fatalf("update: err = %v, want ErrEntryKind", err)
	}
	if _, err := cm.UpsertEntry(ctx, catalog.Entry{Slug: "u", Name: "U", Doc: `{"kind":"place","seats":true,"address":"x"}`}); !errors.Is(err, catalog.ErrEntryKind) {
		t.Fatalf("upsert: err = %v, want ErrEntryKind", err)
	}
}

func TestEntryKind_AnOverrideReplacesTheDefinition(t *testing.T) {
	cm, ctx := newKindedManager(t, placeKinds, []any{
		map[string]any{"name": "place", "fields": []any{map[string]any{"name": "seats", "type": "number", "required": true}}},
	})
	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "o", Name: "O", Doc: `{"kind":"place","seats":2}`}); err != nil {
		t.Fatalf("create under override: %v", err)
	}
	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "p", Name: "P", Doc: `{"kind":"place","address":"1 Lane"}`}); !errors.Is(err, catalog.ErrEntryKind) {
		t.Fatalf("create under override: err = %v, want ErrEntryKind", err)
	}
}

func TestEntryKind_AMalformedSettingIsRefused(t *testing.T) {
	cm, ctx := newKindedManager(t, []any{map[string]any{"name": "place", "fields": []any{map[string]any{"name": "x", "type": "date"}}}}, nil)
	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "m", Name: "M", Doc: `{"kind":"place"}`}); err == nil || errors.Is(err, catalog.ErrEntryKind) {
		t.Fatalf("create under a malformed setting: err = %v, want a setting error", err)
	}
}

func slugOf(s string) string {
	out := []rune{}
	for _, r := range s {
		if r == ' ' {
			r = '-'
		}
		out = append(out, r)
	}
	return string(out)
}
