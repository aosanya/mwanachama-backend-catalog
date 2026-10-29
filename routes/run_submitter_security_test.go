package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/routes"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const mountedCaller = "the-signed-in-caller"

func newMountedServer(t *testing.T) (http.Handler, catalog.CatalogManager, []routes.Tool) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s, err := catalog.LoadSpec(filepath.Join("..", "spec", "examples", "agency.catalog.json"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	m := routes.Mount{Caller: func(context.Context) string { return mountedCaller }}
	mux := http.NewServeMux()
	for _, r := range routes.RoutesFor(cm, m) {
		mux.Handle(r.Method+" "+r.Path, r.Handler)
	}
	return mux, cm, routes.ToolsFor(cm, m)
}

func toolNamed(t *testing.T, tools []routes.Tool, name string) func(context.Context, json.RawMessage) (any, error) {
	t.Helper()
	for _, tl := range tools {
		if tl.Name == name {
			return tl.Invoke
		}
	}
	t.Fatalf("precondition: no tool named %s is built", name)
	return nil
}

func TestCAT13_RunSubmitterIsNeverTakenFromTheRequestBody(t *testing.T) {
	h, _, tools := newMountedServer(t)

	const claimed = "someone-else-entirely"
	recorded := do(t, h, "POST", "/runs", `{"run_ref":"r1","submitted_by":"`+claimed+`","templates_filed":1,"summary":"s"}`)
	if recorded.Code != http.StatusCreated {
		t.Fatalf("precondition: recording a run answered %d %s", recorded.Code, recorded.Body.String())
	}

	record := toolNamed(t, tools, "catalog_run_record")
	_, err := record(context.Background(), json.RawMessage(`{"run_ref":"r2","submitted_by":"`+claimed+`","templates_filed":1,"summary":"s"}`))
	if err == nil || !strings.Contains(err.Error(), `"submitted_by"`) {
		t.Errorf("catalog_run_record accepted a caller-supplied submitter; want a refusal naming \"submitted_by\", got err=%v", err)
	}
	if _, err := record(context.Background(), json.RawMessage(`{"run_ref":"r2","templates_filed":1,"summary":"s"}`)); err != nil {
		t.Fatalf("precondition: recording a run over MCP without a submitter failed: %v", err)
	}

	listed := do(t, h, "GET", "/runs", "")
	var runs []catalog.MiningRun
	if err := json.Unmarshal(listed.Body.Bytes(), &runs); err != nil || len(runs) != 2 {
		t.Fatalf("precondition: expected two stored runs, got %d %s", listed.Code, listed.Body.String())
	}
	for _, run := range runs {
		if run.SubmittedBy != mountedCaller {
			t.Errorf("run %q recorded submitted_by %q; want the mount's caller %q", run.RunRef, run.SubmittedBy, mountedCaller)
		}
	}
}
