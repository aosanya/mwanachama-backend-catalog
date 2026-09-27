package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT13_OpenBug_RunSubmittedByTakenFromRequestBody(t *testing.T) {
	h, _, _ := newServer(t)

	const claimed = "someone-else-entirely"
	res := do(t, h, "POST", "/runs", `{"run_id":"r-1","submitted_by":"`+claimed+`","templates_filed":3,"summary":"ok"}`)
	if res.Code != http.StatusCreated {
		t.Fatalf("precondition: recording a run answered %d %s", res.Code, res.Body.String())
	}

	var stored catalog.MiningRun
	if err := json.Unmarshal(res.Body.Bytes(), &stored); err != nil {
		t.Fatalf("precondition: decode response: %v", err)
	}
	if stored.SubmittedBy == claimed {
		t.Errorf("CAT13 open bug: the stored run's submitted_by is %q, the value the request body claimed; "+
			"it should record the caller that actually reported the run, never a value the caller supplied "+
			"(the same class CAT12 already closed for issue_share_link)", stored.SubmittedBy)
	}
}
