package mwanachamacatalog_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func managerDeclaring(t *testing.T, roles ...string) (catalog.CatalogManager, context.Context) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(".", "spec", "examples", "agency.catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var declared map[string]any
	if err := json.Unmarshal(raw, &declared); err != nil {
		t.Fatal(err)
	}
	keep := map[string]bool{}
	for _, r := range roles {
		keep[r] = true
	}
	var objects []any
	for _, o := range declared["objects"].([]any) {
		if role, _ := o.(map[string]any)["role"].(string); keep[role] {
			objects = append(objects, o)
		}
	}
	declared["objects"] = objects
	narrowed, err := json.Marshal(declared)
	if err != nil {
		t.Fatal(err)
	}
	s, err := catalog.ParseSpec(narrowed)
	if err != nil {
		t.Fatalf("parse narrowed spec: %v", err)
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("a manager over only %v must build: %v", roles, err)
	}
	return cm, context.Background()
}

func entryOperationsWork(t *testing.T, cm catalog.CatalogManager, ctx context.Context) {
	t.Helper()
	seed(t, cm, ctx, "open-one", catalog.VisibilityPublic)
	seed(t, cm, ctx, "closed-one", catalog.VisibilityPrivate)
	if _, err := cm.GetEntry(ctx, "open-one"); err != nil {
		t.Fatalf("get entry: %v", err)
	}
	listed, err := cm.ListPublicEntries(ctx, catalog.ListFilter{})
	if err != nil || len(listed) != 1 {
		t.Fatalf("public listing = %v (%v)", listed, err)
	}
	_, key, err := cm.IssueShareLink(ctx, "closed-one", "board", "operator")
	if err != nil {
		t.Fatalf("issue share link: %v", err)
	}
	if _, err := cm.OpenEntry(ctx, "closed-one", key); err != nil {
		t.Fatalf("open private entry by key: %v", err)
	}
	if err := cm.DeleteEntry(ctx, "open-one", ""); err != nil {
		t.Fatalf("delete entry: %v", err)
	}
}

func TestAManagerOverTheRegistersRolesServesEntriesAndSuggestions(t *testing.T) {
	cm, ctx := managerDeclaring(t, "entry", "entry_private", "share_link", "setting", "suggestion", "suggestion_match")
	entryOperationsWork(t, cm, ctx)

	if _, err := cm.PutSetting(ctx, "k", "v"); err != nil {
		t.Fatalf("put setting: %v", err)
	}
	sg, err := cm.FileSuggestion(ctx, catalog.Suggestion{Slug: "wanted", Title: "Wanted"})
	if err != nil {
		t.Fatalf("file suggestion: %v", err)
	}
	if _, err := cm.DecideSuggestion(ctx, sg.ID, catalog.SuggestionStatus("approved"), intPtr(1)); err != nil {
		t.Fatalf("approve suggestion: %v", err)
	}
	if _, err := cm.RecordSuggestionAttempt(ctx, sg.ID, "run-1", "did not build"); err != nil {
		t.Fatalf("an attempt must count with no comment log declared: %v", err)
	}
	if _, err := cm.AttachSuggestion(ctx, sg.ID, "closed-one", "matches"); err != nil {
		t.Fatalf("attach suggestion: %v", err)
	}
	if _, err := cm.CommentOnSuggestion(ctx, sg.ID, catalog.CommentAuthor("admin"), "hi"); !errors.Is(err, catalog.ErrRoleNotDeclared) {
		t.Fatalf("comment with no comment role = %v, want ErrRoleNotDeclared", err)
	}
	if _, err := cm.RecordRun(ctx, catalog.MiningRun{}); !errors.Is(err, catalog.ErrRoleNotDeclared) {
		t.Fatalf("record run = %v, want ErrRoleNotDeclared", err)
	}
	if _, err := cm.ListRuns(ctx, 0); !errors.Is(err, catalog.ErrRoleNotDeclared) {
		t.Fatalf("list runs = %v, want ErrRoleNotDeclared", err)
	}
	if _, err := cm.ListPageViews(ctx); !errors.Is(err, catalog.ErrRoleNotDeclared) {
		t.Fatalf("list page views = %v, want ErrRoleNotDeclared", err)
	}
	if err := cm.RecordPageView(ctx, catalog.PageKind("index"), ""); !errors.Is(err, catalog.ErrRoleNotDeclared) {
		t.Fatalf("record page view = %v, want ErrRoleNotDeclared", err)
	}
}

func TestAManagerWithNoSettingOrSuggestionRolesServesEntriesAndRefusesTheRestByName(t *testing.T) {
	cm, ctx := managerDeclaring(t, "entry", "entry_private", "share_link")
	entryOperationsWork(t, cm, ctx)

	if _, err := cm.PutSetting(ctx, "k", "v"); !errors.Is(err, catalog.ErrRoleNotDeclared) {
		t.Fatalf("put setting = %v, want ErrRoleNotDeclared", err)
	}
	_, err := cm.GetSetting(ctx, "k")
	if !errors.Is(err, catalog.ErrRoleNotDeclared) || !errors.Is(err, catalog.ErrSettingNotFound) {
		t.Fatalf("get setting = %v, want ErrRoleNotDeclared and ErrSettingNotFound", err)
	}
	if _, err := cm.FileSuggestion(ctx, catalog.Suggestion{Slug: "wanted", Title: "Wanted"}); !errors.Is(err, catalog.ErrRoleNotDeclared) {
		t.Fatalf("file suggestion = %v, want ErrRoleNotDeclared", err)
	}
	if _, err := cm.ListSuggestions(ctx); !errors.Is(err, catalog.ErrRoleNotDeclared) {
		t.Fatalf("list suggestions = %v, want ErrRoleNotDeclared", err)
	}
	if _, err := cm.ListEntryMatches(ctx, "closed-one"); !errors.Is(err, catalog.ErrRoleNotDeclared) {
		t.Fatalf("list entry matches = %v, want ErrRoleNotDeclared", err)
	}
}
