package mwanachamacatalog

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"gorm.io/gorm"
)

const libraryExamplePath = "spec/examples/library.catalog.json"

func examplePaths(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("spec", "examples", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("glob examples: %v (%d found)", err, len(paths))
	}
	sort.Strings(paths)
	return paths
}

func tableNames(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var names []string
	if err := db.Raw(`select name from sqlite_master where type='table' and name not like 'sqlite_%'`).
		Scan(&names).Error; err != nil {
		t.Fatalf("list tables: %v", err)
	}
	sort.Strings(names)
	return names
}

// The same module under two domains, in one database. The spec engine's own
// tests prove this over fixtures of their own; this proves it over the specs
// this module actually ships.
func TestShippedExamplesCoexist(t *testing.T) {
	db := openTestDB(t)

	agency := loadExampleSpec(t)
	library, err := LoadSpec(libraryExamplePath)
	if err != nil {
		t.Fatalf("load library: %v", err)
	}

	for _, s := range []*spec.Spec{agency, library} {
		if err := spec.Migrate(db, s); err != nil {
			t.Fatalf("migrate %s: %v", s.Domain, err)
		}
	}

	want := []string{
		"library_catalog_acquisition_notes", "library_catalog_copies", "library_catalog_import_runs",
		"library_catalog_page_views", "library_catalog_preview_links",
		"library_catalog_purchase_requests", "library_catalog_request_fulfilments",
		"library_catalog_request_notes", "library_catalog_settings", "library_catalog_titles",
		"wakala_catalog_agencies", "wakala_catalog_agency_privates", "wakala_catalog_mining_runs",
		"wakala_catalog_page_views", "wakala_catalog_settings", "wakala_catalog_share_links",
		"wakala_catalog_suggestion_comments", "wakala_catalog_suggestion_matches",
		"wakala_catalog_suggestions",
	}
	if got := tableNames(t, db); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("tables =\n  %v\nwant\n  %v", got, want)
	}

	for _, tc := range []struct {
		s         *spec.Spec
		wantName  string
		wantTable string
	}{
		{agency, "agency", "wakala_catalog_agencies"},
		{library, "title", "library_catalog_titles"},
	} {
		o, ok := tc.s.ByRole(roleEntry)
		if !ok {
			t.Fatalf("%s: no object fills the entry role", tc.s.Domain)
		}
		if o.Name != tc.wantName || tc.s.TableFor(o) != tc.wantTable {
			t.Errorf("%s: entry is %q at %q, want %q at %q",
				tc.s.Domain, o.Name, tc.s.TableFor(o), tc.wantName, tc.wantTable)
		}
	}

	if _, ok := library.Object("copy"); !ok {
		t.Error("the library declares a copy object the agency does not")
	}
	if _, ok := agency.Object("copy"); ok {
		t.Error("the agency should not have a copy object")
	}
}

// Every shipped example fills every role this module's rules reach for. A
// spec that does not should stop the process starting, not fail a request.
func TestShippedExamplesFillEveryRole(t *testing.T) {
	needed := []string{
		roleEntry, roleEntryPrivate, roleShareLink, roleSuggestion,
		roleSuggestionComment, roleSuggestionMatch, roleRun, roleSetting, rolePageView,
	}
	for _, path := range examplePaths(t) {
		s, err := LoadSpec(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
		if err := s.RequireRoles(needed...); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}

// A process runs the migration at every start, so running it twice has to be
// a no-op.
func TestShippedExampleMigratesIdempotently(t *testing.T) {
	db := openTestDB(t)
	s := loadExampleSpec(t)
	for i := 0; i < 3; i++ {
		if err := spec.Migrate(db, s); err != nil {
			t.Fatalf("migrate %d: %v", i, err)
		}
	}
	if n := len(tableNames(t, db)); n != 9 {
		t.Errorf("tables = %d, want 9", n)
	}
}

// A domain's own vocabulary becomes an index without this module learning
// the word: the agency indexes a sector, the library an ISBN.
func TestShippedIndexesComeFromTheDomain(t *testing.T) {
	for _, tc := range []struct{ path, want string }{
		{examplePath, "json_extract(doc, '$._template.sector')"},
		{libraryExamplePath, "json_extract(doc, '$.identifiers.isbn')"},
	} {
		s, err := LoadSpec(tc.path)
		if err != nil {
			t.Fatalf("load %s: %v", tc.path, err)
		}
		if !strings.Contains(strings.Join(s.DDL("sqlite"), "\n"), tc.want) {
			t.Errorf("%s: DDL does not index %s", tc.path, tc.want)
		}
	}

	s := loadExampleSpec(t)
	pg := strings.Join(s.DDL("postgres"), "\n")
	if !strings.Contains(pg, `doc #>> '{_template,sector}'`) {
		t.Error("postgres DDL does not use the jsonb path operator")
	}
	if !strings.Contains(pg, "doc jsonb") {
		t.Error("postgres DDL does not declare doc as jsonb")
	}
}

// Two of this module's objects are keyed by a pair rather than an id.
func TestShippedCompositePrimaryKeys(t *testing.T) {
	ddl := strings.Join(loadExampleSpec(t).DDL("sqlite"), "\n")
	for _, want := range []string{"primary key (suggestion_id, slug)", "primary key (page, slug)"} {
		if !strings.Contains(ddl, want) {
			t.Errorf("DDL is missing %q", want)
		}
	}
}

// A required field must not carry a default: the default is exactly what
// would stop an omitted value being noticed.
func TestShippedExamplesHaveNoDefaultedRequiredColumn(t *testing.T) {
	for _, path := range examplePaths(t) {
		s, err := LoadSpec(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
		for _, stmt := range s.DDL("postgres") {
			for _, line := range strings.Split(stmt, "\n") {
				line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), ","))
				if strings.Contains(line, "not null") && strings.Contains(line, "default") {
					t.Errorf("%s: column is both required and defaulted: %q", path, line)
				}
			}
		}
	}
}

// Every object and field in the shipped examples says something, rather than
// restating its own name. The floor is a sentence, not a character count.
func TestShippedExamplesAreDescribed(t *testing.T) {
	sentence := func(s string) bool { return len(strings.Fields(s)) >= 3 }
	restates := func(desc, name string) bool {
		d := strings.ToLower(strings.Trim(desc, " ."))
		return d == strings.ReplaceAll(name, "_", " ") || d == name
	}

	for _, path := range examplePaths(t) {
		s, err := LoadSpec(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
		for _, o := range s.Objects {
			if !sentence(o.Description) || restates(o.Description, o.Name) {
				t.Errorf("%s: object %q is not described: %q", path, o.Name, o.Description)
			}
			for _, f := range o.Fields {
				if !sentence(f.Description) || restates(f.Description, f.Name) {
					t.Errorf("%s: %s.%s is not described: %q", path, o.Name, f.Name, f.Description)
				}
			}
		}
	}
}

// Postgres requires an expression index's expression in its own parentheses:
// `on t ((expr))`. A single pair is a syntax error there and accepted by
// SQLite, so only an explicit assertion catches it.
func TestShippedExpressionIndexesAreDoubleParenthesised(t *testing.T) {
	for _, path := range examplePaths(t) {
		s, err := LoadSpec(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
		for _, dialect := range []string{"postgres", "sqlite"} {
			for _, stmt := range s.DDL(dialect) {
				if !strings.Contains(stmt, "create index") && !strings.Contains(stmt, "create unique index") {
					continue
				}
				isExpr := strings.Contains(stmt, "#>>") || strings.Contains(stmt, "json_extract")
				if isExpr && !strings.Contains(stmt, "((") {
					t.Errorf("%s/%s: expression index is not double-parenthesised:\n  %s", path, dialect, stmt)
				}
			}
		}
	}
}
