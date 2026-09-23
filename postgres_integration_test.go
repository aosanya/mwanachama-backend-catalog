//go:build integration

package mwanachamacatalog_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/spec"
)

const prettyDoc = `{
  "agency": {
    "name": "Pretty"
  },
  "_template": {
    "sector": "health"
  }
}`

const farmingDoc = `{"agency":{"name":"Farming"},"_template":{"sector":"farming"}}`

func randomInstance(t *testing.T, length int) string {
	t.Helper()
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random instance: %v", err)
	}
	return ("pg" + hex.EncodeToString(b))[:length]
}

func instanceBudget(t *testing.T, s *spec.Spec) int {
	t.Helper()
	probe := *s
	probe.Instance = "x"
	longest := 0
	for _, o := range probe.Objects {
		if n := len(probe.TableFor(o)); n > longest {
			longest = n
		}
		for _, idx := range o.Indexes {
			if n := len(probe.IndexFor(o, idx)); n > longest {
				longest = n
			}
		}
	}
	return spec.MaxIdentifier - (longest - 1)
}

func postgresSpec(t *testing.T) *spec.Spec {
	t.Helper()
	s, err := catalog.LoadSpec(filepath.Join(".", "spec", "examples", "agency.catalog.json"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}

	budget := instanceBudget(t, s)
	if budget < 6 {
		t.Fatalf("the shipped spec leaves only %d bytes for an instance, too few to name one per run", budget)
	}
	s.Instance = randomInstance(t, budget)
	if err := s.Validate(); err != nil {
		t.Fatalf("spec is unusable at instance %q: %v", s.Instance, err)
	}
	return s
}

func newPostgresDB(t *testing.T) (*gorm.DB, *spec.Spec) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		t.Skip("POSTGRES_URL not set; skipping Postgres integration test")
	}
	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}

	s := postgresSpec(t)
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		for _, o := range s.Objects {
			db.Exec("drop table if exists " + s.TableFor(o) + " cascade")
		}
	})
	return db, s
}

func newPostgresManager(t *testing.T) (catalog.CatalogManager, *gorm.DB, *spec.Spec, context.Context) {
	t.Helper()
	db, s := newPostgresDB(t)
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	return cm, db, s, context.Background()
}

func entryObject(t *testing.T, s *spec.Spec) spec.Object {
	t.Helper()
	o, ok := s.ByRole("entry")
	if !ok {
		t.Fatal(`the shipped spec fills no object for the role "entry"`)
	}
	return o
}

func TestPostgresDocumentColumnIsJSONB(t *testing.T) {
	db, s := newPostgresDB(t)

	var dataType string
	err := db.Raw(
		`select data_type from information_schema.columns
		 where table_schema = current_schema() and table_name = ? and column_name = 'doc'`,
		s.TableFor(entryObject(t, s)),
	).Scan(&dataType).Error
	if err != nil {
		t.Fatalf("read column type: %v", err)
	}
	if dataType != "jsonb" {
		t.Fatalf("doc is %q on Postgres, want jsonb", dataType)
	}
}

func TestPostgresListEntriesFiltersOnADocumentPath(t *testing.T) {
	cm, _, _, ctx := newPostgresManager(t)

	for _, e := range []catalog.Entry{
		{Slug: "health-one", Name: "Health One", Summary: "a summary", Doc: prettyDoc},
		{Slug: "farming-one", Name: "Farming One", Summary: "a summary", Doc: farmingDoc},
	} {
		if _, err := cm.CreateEntry(ctx, e); err != nil {
			t.Fatalf("create %s: %v", e.Slug, err)
		}
		if _, err := cm.SetVisibility(ctx, e.Slug, catalog.VisibilityPublic); err != nil {
			t.Fatalf("publish %s: %v", e.Slug, err)
		}
	}

	got, err := cm.ListEntries(ctx, catalog.ListFilter{
		Docs: []catalog.DocFilter{{Path: "_template.sector", Value: "health"}},
	})
	if err != nil {
		t.Fatalf("list by document path: %v", err)
	}
	if len(got) != 1 || got[0].Slug != "health-one" {
		t.Fatalf("filtering on _template.sector returned %d rows (%v), want just health-one", len(got), slugsOf(got))
	}
}

func slugsOf(entries []catalog.Entry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Slug)
	}
	return out
}

func TestPostgresAcceptsEveryExpressionIndex(t *testing.T) {
	db, s := newPostgresDB(t)

	var created int
	for _, o := range s.Objects {
		for _, idx := range o.Indexes {
			if idx.Path == nil {
				continue
			}
			created++
			var n int64
			err := db.Raw(
				`select count(*) from pg_indexes
				 where schemaname = current_schema() and tablename = ? and indexname = ?`,
				s.TableFor(o), s.IndexFor(o, idx),
			).Scan(&n).Error
			if err != nil {
				t.Fatalf("read pg_indexes: %v", err)
			}
			if n != 1 {
				t.Errorf("expression index %s was not created on %s", s.IndexFor(o, idx), s.TableFor(o))
			}
		}
	}
	if created == 0 {
		t.Fatal("the shipped spec declares no path index, so this proves nothing")
	}
}

func TestPostgresRefusesASinglyParenthesisedExpressionIndex(t *testing.T) {
	db, s := newPostgresDB(t)

	var mutated string
	for _, stmt := range s.DDL("postgres") {
		if !strings.Contains(stmt, "#>>") {
			continue
		}
		mutated = strings.Replace(stmt, "((", "(", 1)
		mutated = strings.Replace(mutated, "))", ")", 1)
		mutated = strings.Replace(mutated, "_idx on ", "_mutated_idx on ", 1)
		break
	}
	if mutated == "" {
		t.Fatal("no expression index in the emitted DDL to mutate")
	}

	if err := db.Exec(mutated).Error; err == nil {
		db.Exec(strings.Replace(mutated, "create index if not exists ", "drop index if exists ", 1))
		t.Fatalf("Postgres accepted a singly-parenthesised expression index, so the double parentheses are untested:\n  %s", mutated)
	}
}

func TestPostgresTruncatesLongIdentifiersSilently(t *testing.T) {
	db, _ := newPostgresDB(t)

	shared := randomInstance(t, 12)
	shared += strings.Repeat("x", spec.MaxIdentifier-len(shared))
	first := shared + "_alpha"
	second := shared + "_beta"
	t.Cleanup(func() { db.Exec("drop table if exists " + shared + " cascade") })

	if err := db.Exec("create table " + first + " (alpha text)").Error; err != nil {
		t.Fatalf("create the first table: %v", err)
	}
	if err := db.Exec("create table if not exists " + second + " (beta text)").Error; err != nil {
		t.Fatalf("create the second table: %v", err)
	}

	var columns []string
	if err := db.Raw(
		`select column_name from information_schema.columns
		 where table_schema = current_schema() and table_name = ?`, shared,
	).Scan(&columns).Error; err != nil {
		t.Fatalf("read columns: %v", err)
	}
	if len(columns) != 1 || columns[0] != "alpha" {
		t.Fatalf("%s holds %v; the two names were expected to truncate onto one relation carrying only alpha", shared, columns)
	}
}

func TestPostgresDocumentSurvivesJSONBSemantically(t *testing.T) {
	cm, _, _, ctx := newPostgresManager(t)

	if _, err := cm.CreateEntry(ctx, catalog.Entry{
		Slug: "pretty", Name: "Pretty", Summary: "a summary", Doc: prettyDoc,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := cm.GetEntry(ctx, "pretty")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	var sent, stored any
	if err := json.Unmarshal([]byte(prettyDoc), &sent); err != nil {
		t.Fatalf("parse the sent document: %v", err)
	}
	if err := json.Unmarshal([]byte(got.Doc), &stored); err != nil {
		t.Fatalf("parse the stored document: %v", err)
	}
	if !reflect.DeepEqual(sent, stored) {
		t.Fatalf("the document changed meaning through jsonb:\n sent:   %s\n stored: %s", prettyDoc, got.Doc)
	}
}

func TestPostgresResendingAnUnchangedDocumentKeepsItsReview(t *testing.T) {
	cm, _, _, ctx := newPostgresManager(t)

	e := catalog.Entry{Slug: "resent", Name: "Resent", Summary: "a summary", Doc: prettyDoc}
	if _, err := cm.CreateEntry(ctx, e); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := cm.SetVisibility(ctx, "resent", catalog.VisibilityPublic); err != nil {
		t.Fatalf("publish: %v", err)
	}

	again, err := cm.UpsertEntry(ctx, e)
	if err != nil {
		t.Fatalf("re-send: %v", err)
	}
	if again.Visibility != catalog.VisibilityPublic {
		t.Fatalf("re-sending a byte-identical document returned a published Entry to %q; jsonb normalised what was stored, so the comparison in UpsertEntry saw a change that was not one", again.Visibility)
	}
}
