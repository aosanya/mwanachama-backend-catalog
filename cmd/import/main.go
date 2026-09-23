// Command import fills a catalog from a directory of mined agency folders —
// the shape publish_agency.py already writes: one folder per agency holding
// agency.json, README.md, sales.md and any per-module config JSON beside
// them. The folder name is the slug.
//
// It is re-runnable. Each agency goes through UpsertEntry, so an unchanged
// document leaves its review alone and a changed one returns to pending.
//
//	import -dir <agencies> -dsn <postgres-url> [-instance catalog] [-dry-run] [-limit N]
//
// With no -dsn it reads and validates every folder without writing, which is
// the cheapest way to find a malformed document before touching a database.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

type folder struct {
	slug    string
	entry   catalog.Entry
	private catalog.EntryPrivate
}

func main() {
	dir := flag.String("dir", "", "directory of agency folders (required)")
	dsn := flag.String("dsn", "", "Postgres URL; empty means validate only")
	instance := flag.String("instance", "catalog", "table-name prefix")
	dryRun := flag.Bool("dry-run", false, "read and validate, never write")
	limit := flag.Int("limit", 0, "import at most N agencies (0 = all)")
	flag.Parse()

	if *dir == "" {
		fail("-dir is required")
	}

	folders, problems := readAll(*dir, *limit)
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, "skipped:", p)
	}
	fmt.Printf("read %d agencies from %s (%d skipped)\n", len(folders), *dir, len(problems))

	if *dsn == "" || *dryRun {
		fmt.Println("validate only — nothing written")
		if len(problems) > 0 {
			os.Exit(1)
		}
		return
	}

	db, err := gorm.Open(postgres.Open(*dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fail("connect: %v", err)
	}
	tables := catalog.DefaultTableNames(*instance)
	if err := catalog.Migrate(db, tables); err != nil {
		fail("migrate: %v", err)
	}
	cm, err := catalog.NewCatalogManager(db, tables)
	if err != nil {
		fail("manager: %v", err)
	}

	ctx := context.Background()
	var written, failed int
	for _, f := range folders {
		if _, err := cm.UpsertEntry(ctx, f.entry); err != nil {
			fmt.Fprintf(os.Stderr, "%s: entry: %v\n", f.slug, err)
			failed++
			continue
		}
		if _, err := cm.PutPrivate(ctx, f.private); err != nil {
			fmt.Fprintf(os.Stderr, "%s: private: %v\n", f.slug, err)
			failed++
			continue
		}
		written++
	}
	fmt.Printf("wrote %d, failed %d, into %s_*\n", written, failed, *instance)
	if failed > 0 || len(problems) > 0 {
		os.Exit(1)
	}
}

// readAll loads every agency folder under dir, returning what parsed and a
// line per folder that did not.
func readAll(dir string, limit int) ([]folder, []string) {
	names, err := os.ReadDir(dir)
	if err != nil {
		fail("read %s: %v", dir, err)
	}
	sort.Slice(names, func(i, j int) bool { return names[i].Name() < names[j].Name() })

	var out []folder
	var problems []string
	for _, n := range names {
		if !n.IsDir() {
			continue
		}
		if limit > 0 && len(out) >= limit {
			break
		}
		f, err := readFolder(dir, n.Name())
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", n.Name(), err))
			continue
		}
		out = append(out, f)
	}
	return out, problems
}

func readFolder(dir, name string) (folder, error) {
	path := filepath.Join(dir, name)
	raw, err := os.ReadFile(filepath.Join(path, "agency.json"))
	if err != nil {
		return folder{}, fmt.Errorf("no agency.json")
	}

	// Only the three domain-neutral card fields are lifted out. The
	// agency's sector and archetype stay in `_template` inside the
	// document, where a filter reads them by path.
	var doc struct {
		Agency struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"agency"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return folder{}, fmt.Errorf("agency.json does not parse: %w", err)
	}
	if doc.Agency.Name == "" {
		return folder{}, fmt.Errorf("agency.json has no agency.name")
	}

	slug := slugify(name)
	if !catalogSlug(slug) {
		return folder{}, fmt.Errorf("folder name yields no usable slug")
	}

	entry := catalog.Entry{
		Slug:        slug,
		Name:        doc.Agency.Name,
		Summary:     doc.Agency.Description,
		Doc:         string(raw),
		Visibility:  catalog.VisibilityPending,
		SubmittedBy: "import",
	}
	if !entry.Validate() {
		return folder{}, fmt.Errorf("entry does not validate")
	}

	configs, err := readConfigs(path)
	if err != nil {
		return folder{}, err
	}
	private := catalog.EntryPrivate{
		Slug:          slug,
		Readme:        readFile(filepath.Join(path, "README.md")),
		Sales:         readFile(filepath.Join(path, "sales.md")),
		ModuleConfigs: configs,
	}
	if !private.Validate() {
		return folder{}, fmt.Errorf("private parts do not validate")
	}
	return folder{slug: slug, entry: entry, private: private}, nil
}

// readConfigs folds every JSON file beside agency.json into one object
// keyed by the module code the filename carries.
func readConfigs(path string) (string, error) {
	names, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	configs := map[string]json.RawMessage{}
	for _, n := range names {
		fn := n.Name()
		if n.IsDir() || fn == "agency.json" || !strings.HasSuffix(fn, ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(path, fn))
		if err != nil {
			return "", err
		}
		if !json.Valid(raw) {
			return "", fmt.Errorf("%s does not parse", fn)
		}
		configs[strings.TrimSuffix(fn, ".json")] = raw
	}
	if len(configs) == 0 {
		return "", nil
	}
	out, err := json.Marshal(configs)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

// slugify folds a folder name into a slug the same way publish_agency.py
// does, so a folder imported here takes the slug it would have been
// published under.
var notSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	return strings.Trim(notSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

func catalogSlug(s string) bool {
	e := catalog.Entry{Slug: s, Name: "x", Doc: "{}", Visibility: catalog.VisibilityPending}
	return e.Validate()
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
