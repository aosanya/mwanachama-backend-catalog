package spec_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aosanya/mwanachama-backend-catalog/spec"
)

func open(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return db
}

func load(t *testing.T, path string) *spec.Spec {
	t.Helper()
	s, err := spec.Load(path)
	if err != nil {
		t.Fatalf("load %s: %v", path, err)
	}
	return s
}

func tables(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var names []string
	if err := db.Raw(`select name from sqlite_master where type='table' and name not like 'sqlite_%'`).
		Scan(&names).Error; err != nil {
		t.Fatalf("list tables: %v", err)
	}
	sort.Strings(names)
	return names
}

// Two domains, one module, one database: the whole point of declaring
// objects rather than writing them. Neither spec changes a line of Go.
func TestTwoDomainsCoexist(t *testing.T) {
	db := open(t)

	agency := load(t, "examples/agency.catalog.json")
	library := load(t, "examples/library.catalog.json")

	if err := spec.Migrate(db, agency); err != nil {
		t.Fatalf("migrate agency: %v", err)
	}
	if err := spec.Migrate(db, library); err != nil {
		t.Fatalf("migrate library: %v", err)
	}

	got := tables(t, db)
	want := []string{
		"agency_agencies", "agency_agency_privates", "agency_mining_runs",
		"agency_page_views", "agency_settings", "agency_share_links",
		"agency_suggestion_comments", "agency_suggestion_matches", "agency_suggestions",
		"library_acquisition_notes", "library_copies", "library_import_runs",
		"library_page_views", "library_preview_links", "library_purchase_requests",
		"library_request_fulfilments", "library_request_notes", "library_settings",
		"library_titles",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("tables =\n  %v\nwant\n  %v", got, want)
	}

	// The same object is a different table in each domain, and the library
	// has one the agency does not.
	e, _ := agency.Object("entry")
	if agency.TableFor(e) != "agency_agencies" {
		t.Errorf("agency entry table = %q", agency.TableFor(e))
	}
	e, _ = library.Object("entry")
	if library.TableFor(e) != "library_titles" {
		t.Errorf("library entry table = %q", library.TableFor(e))
	}
	if _, ok := library.Object("copy"); !ok {
		t.Error("library declares a copy object the agency does not")
	}
	if _, ok := agency.Object("copy"); ok {
		t.Error("the agency should not have a copy object")
	}
}

// Migrating twice must be a no-op, because a process runs it at every start.
func TestMigrateIsIdempotent(t *testing.T) {
	db := open(t)
	s := load(t, "examples/agency.catalog.json")

	for i := 0; i < 3; i++ {
		if err := spec.Migrate(db, s); err != nil {
			t.Fatalf("migrate %d: %v", i, err)
		}
	}
	if n := len(tables(t, db)); n != 9 {
		t.Errorf("tables = %d, want 9", n)
	}
}

// A domain's own vocabulary becomes an index without the module learning the
// word: the agency indexes a sector, the library a subject and an ISBN.
func TestIndexesComeFromTheDomain(t *testing.T) {
	for _, tc := range []struct {
		path string
		want string
	}{
		{"examples/agency.catalog.json", "json_extract(doc, '$._template.sector')"},
		{"examples/library.catalog.json", "json_extract(doc, '$.identifiers.isbn')"},
	} {
		s := load(t, tc.path)
		ddl := strings.Join(s.DDL("sqlite"), "\n")
		if !strings.Contains(ddl, tc.want) {
			t.Errorf("%s: DDL does not index %s", tc.path, tc.want)
		}
	}

	// And the same declaration is jsonb on Postgres.
	s := load(t, "examples/agency.catalog.json")
	pg := strings.Join(s.DDL("postgres"), "\n")
	if !strings.Contains(pg, `doc #>> '{_template,sector}'`) {
		t.Error("postgres DDL does not use the jsonb path operator")
	}
	if !strings.Contains(pg, "doc jsonb") {
		t.Error("postgres DDL does not declare doc as jsonb")
	}
	if strings.Contains(strings.Join(s.DDL("sqlite"), "\n"), "jsonb") {
		t.Error("sqlite DDL should not mention jsonb")
	}
}

// A spec is refused when it loads, not when a request arrives.
func TestValidate_RefusesABadSpec(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"no primary key",
			`{"module":"catalog","domain":"d","instance":"i","objects":[{"name":"a","fields":[{"name":"x","type":"string"}]}]}`,
			"no primary key"},
		{"unknown type",
			`{"module":"catalog","domain":"d","instance":"i","objects":[{"name":"a","fields":[{"name":"x","type":"blob","primary":true}]}]}`,
			"unknown type"},
		{"enum with no values",
			`{"module":"catalog","domain":"d","instance":"i","objects":[{"name":"a","fields":[{"name":"x","type":"enum","primary":true}]}]}`,
			"declares no values"},
		{"duplicate table",
			`{"module":"catalog","domain":"d","instance":"i","objects":[
			  {"name":"a","table":"t","fields":[{"name":"x","type":"string","primary":true}]},
			  {"name":"b","table":"t","fields":[{"name":"x","type":"string","primary":true}]}]}`,
			"more than one object"},
		{"index on an unknown field",
			`{"module":"catalog","domain":"d","instance":"i","objects":[{"name":"a","fields":[{"name":"x","type":"string","primary":true}],
			  "indexes":[{"name":"i","fields":["nope"]}]}]}`,
			"unknown field"},
		{"not_deleted without a deleted field",
			`{"module":"catalog","domain":"d","instance":"i","objects":[{"name":"a","fields":[{"name":"x","type":"string","primary":true}],
			  "indexes":[{"name":"i","fields":["x"],"not_deleted":true}]}]}`,
			"no deleted field"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := spec.Parse([]byte(tc.body))
			if err == nil {
				t.Fatal("spec loaded, want a refusal")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

// Names reach the database as SQL text, so a name that could close an
// identifier and continue the statement must be refused outright.
func TestValidate_RefusesNamesThatAreNotNames(t *testing.T) {
	for _, bad := range []string{
		`a"; drop table x; --`,
		"a b",
		"A",
		"_a",
		"a__b",
		"a-",
		"",
		"1a",
	} {
		t.Run(bad, func(t *testing.T) {
			body := `{"module":"catalog","domain":"d","instance":"i","objects":[{"name":"` + bad +
				`","fields":[{"name":"x","type":"string","primary":true}]}]}`
			if _, err := spec.Parse([]byte(body)); err == nil {
				t.Fatalf("object name %q was accepted", bad)
			}
		})
	}
}

// An unknown key is a typo, and a typo that is silently ignored is a field
// nobody notices is missing.
func TestParse_RefusesUnknownKeys(t *testing.T) {
	body := `{"module":"catalog","domain":"d","instance":"i","objekts":[]}`
	if _, err := spec.Parse([]byte(body)); err == nil {
		t.Fatal("unknown key accepted")
	}
}

// A composite key is declared by marking several fields primary.
func TestCompositePrimaryKey(t *testing.T) {
	s := load(t, "examples/agency.catalog.json")
	ddl := strings.Join(s.DDL("sqlite"), "\n")
	if !strings.Contains(ddl, "primary key (suggestion_id, slug)") {
		t.Error("suggestion_match should have a composite key")
	}
	if !strings.Contains(ddl, "primary key (page, slug)") {
		t.Error("page_view should have a composite key")
	}
}

// A required field must not carry a default: the default is exactly what
// would stop an omitted value being noticed.
func TestRequiredFieldsHaveNoDefault(t *testing.T) {
	s := load(t, "examples/agency.catalog.json")
	for _, stmt := range s.DDL("postgres") {
		for _, line := range strings.Split(stmt, "\n") {
			line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), ","))
			if strings.Contains(line, "not null") && strings.Contains(line, "default") {
				t.Errorf("column is both required and defaulted: %q", line)
			}
		}
	}
}
