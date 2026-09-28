package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestCAT16_OpenHole_FilerTakesAQueuePositionNoDeciderCanSet(t *testing.T) {
	h, _, _ := newServer(t)

	lowest := do(t, h, "POST", "/suggestions", `{"slug":"decided","title":"Decided"}`)
	var decided struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(lowest.Body.Bytes(), &decided); err != nil || lowest.Code != http.StatusCreated {
		t.Fatalf("precondition: filing a plain suggestion answered %d %s", lowest.Code, lowest.Body.String())
	}
	if rec := do(t, h, "POST", "/suggestions/"+decided.ID+"/decide", `{"status":"approved","priority":1}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: a decider placing it at priority 1 answered %d %s", rec.Code, rec.Body.String())
	}

	rec := do(t, h, "POST", "/suggestions", `{"slug":"jumped","title":"Jumped","priority":-7}`)
	if rec.Code >= 400 && rec.Code < 500 {
		return
	}
	var filed struct {
		ID       string `json:"id"`
		Priority int    `json:"priority"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &filed); err != nil {
		t.Fatalf("POST /suggestions answered %d %s", rec.Code, rec.Body.String())
	}
	if filed.Priority < 1 {
		t.Errorf("CAT16 (P2) open hole: a filer holding only catalog.suggestion.file stored priority=%d, ahead of priority 1 — the best position catalog.suggestion.decide can set; got %d %s, want the filing refused 400 or a priority of at least 1", filed.Priority, rec.Code, strings.TrimSpace(rec.Body.String()))
	}
}
