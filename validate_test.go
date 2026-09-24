package mwanachamacatalog

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

// The point of the whole exercise, in one test: the permitted states are
// the spec's, not Go's. A domain that declares a fourth one gets a fourth
// one, with nothing in this package changing — and the same manager built
// on the undeclared spec refuses it.
//
// If the vocabulary were still a map in models/, the first half would fail.
func TestVocabularyComesFromTheSpec(t *testing.T) {
	const embargoed = models.Visibility("embargoed")

	t.Run("undeclared is refused", func(t *testing.T) {
		cm := managerOn(t, loadExampleSpec(t))
		ctx := context.Background()
		mustCreate(t, cm, ctx)

		if _, err := cm.SetVisibility(ctx, "one-entry", embargoed); !errors.Is(err, ErrInvalidEntry) {
			t.Fatalf("err = %v, want ErrInvalidEntry", err)
		}
	})

	t.Run("declared is accepted", func(t *testing.T) {
		cm := managerOn(t, withExtraVisibility(t, string(embargoed)))
		ctx := context.Background()
		mustCreate(t, cm, ctx)

		e, err := cm.SetVisibility(ctx, "one-entry", embargoed)
		if err != nil {
			t.Fatalf("set visibility: %v", err)
		}
		if e.Visibility != embargoed {
			t.Errorf("Visibility = %q, want %q", e.Visibility, embargoed)
		}
	})
}

// A named pattern is the spec's to require and this module's to define, so
// a spec naming one nobody supplies is an error rather than a rule that
// quietly never runs.
func TestCheck_RefusesAPatternNoOneSupplies(t *testing.T) {
	s := loadExampleSpec(t)
	o, _ := s.ByRole(roleEntry)
	for i := range o.Fields {
		if o.Fields[i].Name == "name" {
			o.Fields[i].Matches = "invented"
		}
	}

	err := check(o, models.Entry{Slug: "one-entry", Name: "One", Doc: "{}"})
	if err == nil || !strings.Contains(err.Error(), "does not supply") {
		t.Fatalf("err = %v, want a refusal naming the missing pattern", err)
	}
}

func managerOn(t *testing.T, s *spec.Spec) CatalogManager {
	t.Helper()
	db := openTestDB(t)
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cm, err := NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	return cm
}

func mustCreate(t *testing.T, cm CatalogManager, ctx context.Context) {
	t.Helper()
	if _, err := cm.CreateEntry(ctx, models.Entry{
		Slug: "one-entry", Name: "One", Doc: `{"a":1}`,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
}

func withExtraVisibility(t *testing.T, extra string) *spec.Spec {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(blueprintSource()), &doc); err != nil {
		t.Fatalf("read blueprint: %v", err)
	}
	for _, o := range doc["objects"].([]any) {
		obj := o.(map[string]any)
		if obj["role"] != roleEntry {
			continue
		}
		for _, f := range obj["fields"].([]any) {
			field := f.(map[string]any)
			if field["name"] == "visibility" {
				field["values"] = append(field["values"].([]any), extra)
			}
		}
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("rewrite blueprint: %v", err)
	}
	s, err := loadThrough(t, string(raw))
	if err != nil {
		t.Fatalf("load the example through the mutated blueprint: %v", err)
	}
	return s
}
