package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
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

func TestCAT13_OpenHole_RunSubmitterIsTakenFromTheRequestBody(t *testing.T) {
	h, _, _ := newMountedServer(t)

	const claimed = "someone-else-entirely"
	recorded := do(t, h, "POST", "/runs", `{"run_id":"r1","submitted_by":"`+claimed+`","templates_filed":1,"summary":"s"}`)
	if recorded.Code != http.StatusCreated {
		t.Fatalf("precondition: recording a run answered %d %s", recorded.Code, recorded.Body.String())
	}

	listed := do(t, h, "GET", "/runs", "")
	var runs []catalog.MiningRun
	if err := json.Unmarshal(listed.Body.Bytes(), &runs); err != nil || len(runs) != 1 {
		t.Fatalf("precondition: expected one stored run, got %d %s", listed.Code, listed.Body.String())
	}
	if runs[0].SubmittedBy != mountedCaller {
		t.Errorf("CAT13 (P1) open hole: the stored run's submitted_by is %q, the value the request body claimed; "+
			"it should be %q, the caller the mount supplied, never a value the caller supplied", runs[0].SubmittedBy, mountedCaller)
	}
}
