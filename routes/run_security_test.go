package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/routes"
)

func TestCAT13_OpenHole_RunSubmitterIsTakenFromTheRequestBody(t *testing.T) {
	_, cm, _ := newServer(t)
	mux := http.NewServeMux()
	for _, r := range routes.RoutesFor(cm, routes.Mount{Caller: func(context.Context) string { return "operator-a" }}) {
		mux.Handle(r.Method+" "+r.Path, r.Handler)
	}

	const claimed = "someone-else-entirely"
	recorded := do(t, mux, "POST", "/runs", `{"run_id":"r1","templates_filed":1,"submitted_by":"`+claimed+`"}`)
	if recorded.Code != http.StatusCreated {
		t.Fatalf("precondition: recording a run answered %d %s", recorded.Code, recorded.Body.String())
	}

	listed := do(t, mux, "GET", "/runs", "")
	var runs []catalog.MiningRun
	if err := json.Unmarshal(listed.Body.Bytes(), &runs); err != nil || len(runs) != 1 {
		t.Fatalf("precondition: expected one stored run, got %d %s", listed.Code, listed.Body.String())
	}
	if runs[0].SubmittedBy != "operator-a" {
		t.Errorf("CAT13 (P2) open hole: the stored run's submitted_by is %q, the value the request body claimed; "+
			"it should record the mount's caller %q, never a value the caller supplied", runs[0].SubmittedBy, "operator-a")
	}
}
