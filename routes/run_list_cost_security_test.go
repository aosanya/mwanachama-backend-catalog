package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestCAT32_OpenHole_OneCallerGrowsTheRunListAnswerWithoutBound(t *testing.T) {
	h, _, _ := newServer(t)
	const recorded = 40
	const requestCap = 1 << 20
	body, err := json.Marshal(map[string]any{"run_ref": "flood", "templates_filed": 1, "summary": strings.Repeat("s", 64<<10)})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	stored := 0
	for range recorded {
		if rec := do(t, h, "POST", "/runs", string(body)); rec.Code == http.StatusCreated {
			stored++
		}
	}

	rec := do(t, h, "GET", "/runs", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("precondition: GET /runs answered %d %s", rec.Code, rec.Body.String())
	}
	var listed []json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("precondition: GET /runs did not answer a JSON array: %v", err)
	}
	if got := rec.Body.Len(); got > requestCap {
		t.Errorf("CAT32 (P2): one caller recorded %d of %d run reports of %d bytes each through POST /runs, then one GET /runs with no limit answered %d runs in %d bytes; want the default answer bounded below %d bytes — a default page, a cap on a report's summary, or both — so one caller cannot make a single read cost more than the largest request the mount accepts",
			stored, recorded, len(body), len(listed), got, requestCap)
	}
}
