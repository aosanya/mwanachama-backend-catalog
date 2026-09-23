package mwanachamacatalog

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-catalog/spec"
)

// The column a field lands in is derived from its name, so the rule that
// derives it is worth pinning — especially over an initialism, which is
// where a naive camel-to-snake split scatters letters.
func TestColumnName(t *testing.T) {
	for _, tc := range []struct{ field, want string }{
		{"ID", "id"},
		{"Slug", "slug"},
		{"SubmittedBy", "submitted_by"},
		{"RunID", "run_id"},
		{"SuggestionID", "suggestion_id"},
		{"KeyHash", "key_hash"},
		{"ModuleConfigs", "module_configs"},
		{"TemplatesFiled", "templates_filed"},
		{"MetAt", "met_at"},
	} {
		if got := columnName(tc.field); got != tc.want {
			t.Errorf("columnName(%q) = %q, want %q", tc.field, got, tc.want)
		}
	}
}

// ShareLink.KeyHash is json:"-", so that a stored hash never reaches an API
// response. A codec reading json tags would therefore stop storing it, and
// every share key would silently stop working — the link would be issued,
// the raw key handed over, and nothing would match it.
//
// This is why the join is the field name and not the tag.
func TestEncode_CarriesAFieldThatJSONHides(t *testing.T) {
	s := loadExampleSpec(t)
	o, ok := s.ByRole(roleShareLink)
	if !ok {
		t.Fatal("no object fills the share_link role")
	}

	row, err := encode(o, models.ShareLink{Slug: "x", KeyHash: "deadbeef"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if row["key_hash"] != "deadbeef" {
		t.Errorf("key_hash = %v, want the hash to be stored", row["key_hash"])
	}

	var back models.ShareLink
	if err := decode(o, row, &back); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.KeyHash != "deadbeef" {
		t.Errorf("KeyHash = %q, want it read back", back.KeyHash)
	}
}

// Every declared column is written every time, whether or not the value
// carries anything for it. A map missing a key means "leave it alone" to an
// update, so a field could never be cleared — an Entry would keep an
// ApprovedAt its document changing was supposed to drop.
func TestEncode_WritesEveryDeclaredColumn(t *testing.T) {
	s := loadExampleSpec(t)
	o, _ := s.ByRole(roleEntry)

	row, err := encode(o, models.Entry{Slug: "x", Name: "X"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	for _, f := range o.Fields {
		if _, ok := row[f.Name]; !ok {
			t.Errorf("%q is declared but absent from the row", f.Name)
		}
	}
	if got, ok := row["approved_at"]; !ok || got != "" {
		t.Errorf("approved_at = %v, want an empty value present so it can be cleared", got)
	}
}

func TestNewStore_RefusesASpecThatDisagreesWithTheType(t *testing.T) {
	for _, tc := range []struct{ name, mutate, want string }{
		{"a column no field carries", `{"name":"invented","type":"string","description":"x"},`,
			"which models.Entry does not carry"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := blueprintSource()
			marker := `"fields": [`
			i := strings.Index(raw, marker)
			if i < 0 {
				t.Fatal("the blueprint no longer lays out fields as expected")
			}
			mutated := raw[:i+len(marker)] + tc.mutate + raw[i+len(marker):]

			s, err := loadThrough(t, mutated)
			if err != nil {
				t.Fatalf("the mutated blueprint should still load the example: %v", err)
			}
			_, err = newStore(openTestDB(t), s, map[string]any{roleEntry: models.Entry{}})
			if err == nil {
				t.Fatal("store built, want a refusal")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}

	s := loadExampleSpec(t)
	_, err := newStore(openTestDB(t), s, map[string]any{roleSetting: models.Entry{}})
	if err == nil || !strings.Contains(err.Error(), "does not declare") {
		t.Errorf("err = %v, want a refusal naming the undeclared column", err)
	}
}

const examplePath = "spec/examples/agency.catalog.json"

func loadExampleSpec(t *testing.T) *spec.Spec {
	t.Helper()
	s, err := LoadSpec(examplePath)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	return s
}

func blueprintSource() string { return string(blueprintJSON) }

func loadThrough(t *testing.T, mutatedBlueprint string) (*spec.Spec, error) {
	t.Helper()
	b, err := spec.ParseBlueprint([]byte(mutatedBlueprint))
	if err != nil {
		return nil, err
	}
	return b.Load(examplePath)
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}
