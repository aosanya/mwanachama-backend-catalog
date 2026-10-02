package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestFileSuggestion_AnonymousForgedFieldsInAnyKeyCaseAreTheServers(t *testing.T) {
	operator, anon, _ := municipalityWard(t)

	filed := do(t, anon, http.MethodPost, "/suggestions",
		`{"slug":"case-forged","title":"T","Status":"approved","ATTEMPTS":42,"Id":"chosen","Created_At":"1999-01-01T00:00:00Z","DECIDED_AT":"1999-01-01T00:00:00Z"}`)
	if filed.Code != http.StatusCreated {
		t.Fatalf("precondition: an anonymous POST /suggestions with case-variant keys answered %d %s", filed.Code, filed.Body.String())
	}
	var s catalog.Suggestion
	if err := json.Unmarshal(filed.Body.Bytes(), &s); err != nil || s.ID == "" {
		t.Fatalf("precondition: decode filed suggestion %s: %v", filed.Body.String(), err)
	}

	read := do(t, operator, http.MethodGet, "/suggestions/"+s.ID, "")
	var back catalog.Suggestion
	if err := json.Unmarshal(read.Body.Bytes(), &back); err != nil || read.Code != http.StatusOK {
		t.Fatalf("precondition: the moderator's GET /suggestions/%s answered %d %s", s.ID, read.Code, read.Body.String())
	}
	if back.ID == "chosen" || back.Status != catalog.SuggestionStatusProposed || back.Attempts != 0 ||
		back.CreatedAt == "1999-01-01T00:00:00Z" || back.DecidedAt != "" {
		t.Errorf("a key spelled in another case set a server-owned field on an anonymous filing — the body decodes case-insensitively, so the server must own these whatever the spelling: %+v", back)
	}
}
