package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestListRuns_MalformedLimitIsRefusedWithoutReachingTheStore(t *testing.T) {
	h, cm, ctx := newServer(t)
	for _, id := range []string{"run-a", "run-b", "run-c"} {
		if _, err := cm.RecordRun(ctx, catalog.MiningRun{RunID: id}); err != nil {
			t.Fatalf("RecordRun %s: %v", id, err)
		}
	}

	rec := do(t, h, "GET", "/runs?limit=1", "")
	var one []catalog.MiningRun
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &one) != nil || len(one) != 1 {
		t.Fatalf("precondition: GET /runs?limit=1 answered %d %s, want 200 with exactly one run", rec.Code, rec.Body.String())
	}

	for _, q := range []string{"99999999999999999999", "abc", "1e3", "%00"} {
		rec := do(t, h, "GET", "/runs?limit="+q, "")
		body := strings.ToLower(rec.Body.String())
		if rec.Code != http.StatusBadRequest {
			t.Errorf("GET /runs?limit=%s answered %d, want 400: %s", q, rec.Code, rec.Body.String())
		}
		for _, leak := range []string{"sql", "gorm", "sqlite", "postgres", "strconv", "select"} {
			if strings.Contains(body, leak) {
				t.Errorf("GET /runs?limit=%s refusal names %q, want only the argument: %s", q, leak, rec.Body.String())
			}
		}
	}
}
