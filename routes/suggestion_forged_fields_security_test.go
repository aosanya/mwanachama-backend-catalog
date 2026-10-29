package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestFileSuggestion_ForgedStatusAttemptsIDAndClockAreTheServers(t *testing.T) {
	h, _, _ := newMountedServer(t)

	filed := do(t, h, http.MethodPost, "/suggestions", `{"slug":"forged-one","title":"T","status":"built","attempts":42,"id":"chosen","created_at":"1999-01-01T00:00:00Z","decided_at":"1999-01-01T00:00:00Z"}`)
	if filed.Code != http.StatusCreated {
		t.Fatalf("precondition: POST /suggestions answered %d %s", filed.Code, filed.Body.String())
	}
	var s catalog.Suggestion
	if err := json.Unmarshal(filed.Body.Bytes(), &s); err != nil || s.ID == "" {
		t.Fatalf("precondition: decode filed suggestion %s: %v", filed.Body.String(), err)
	}

	read := do(t, h, http.MethodGet, "/suggestions/"+s.ID, "")
	var back catalog.Suggestion
	if err := json.Unmarshal(read.Body.Bytes(), &back); err != nil || read.Code != http.StatusOK {
		t.Fatalf("GET /suggestions/%s answered %d %s", s.ID, read.Code, read.Body.String())
	}
	if back.ID == "chosen" || back.Status != catalog.SuggestionStatusProposed || back.Attempts != 0 ||
		back.CreatedAt == "1999-01-01T00:00:00Z" || back.DecidedAt != "" {
		t.Errorf("a filed suggestion kept a caller-set field: %+v", back)
	}
	if got := do(t, h, http.MethodGet, "/suggestions/chosen", ""); got.Code != http.StatusNotFound {
		t.Errorf("GET /suggestions/chosen answered %d %s, want 404", got.Code, got.Body.String())
	}

	if got := do(t, h, http.MethodPost, "/suggestions/"+s.ID+"/fulfil", `{"slug":"forged-one","run_ref":"r1"}`); got.Code != http.StatusConflict {
		t.Errorf("fulfilling a proposed suggestion answered %d %s, want 409", got.Code, got.Body.String())
	}
	if got := do(t, h, http.MethodPost, "/suggestions/"+s.ID+"/decide", `{"status":"stalled"}`); got.Code != http.StatusConflict {
		t.Errorf("deciding a suggestion stalled answered %d %s, want 409", got.Code, got.Body.String())
	}
	final := do(t, h, http.MethodGet, "/suggestions/"+s.ID, "")
	var after catalog.Suggestion
	_ = json.Unmarshal(final.Body.Bytes(), &after)
	if after.Status != catalog.SuggestionStatusProposed {
		t.Errorf("after refused transitions the suggestion is %q, want proposed", after.Status)
	}
}
