package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAssigningAClosedBoardItemAnswersConflictThroughTheDeclaredTable(t *testing.T) {
	gated, _, _ := callerMount(t, "clerk-1")
	var b catalog.BoardItem
	_ = json.Unmarshal(sendTo(gated, http.MethodPost, "/board-items", `{"title":"Streetlights out"}`).Body.Bytes(), &b)
	if rec := sendTo(gated, http.MethodPut, "/board-items/"+b.ID+"/assignee", `{"assignee_kind":"actor","assignee_id":"clerk-1"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("assign open = %d %s", rec.Code, rec.Body)
	}
	if rec := sendTo(gated, http.MethodPost, "/board-items/"+b.ID+"/close", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("close = %d %s", rec.Code, rec.Body)
	}
	if rec := sendTo(gated, http.MethodPut, "/board-items/"+b.ID+"/assignee", `{"assignee_kind":"role","assignee_id":"roads-desk"}`); rec.Code != http.StatusConflict {
		t.Fatalf("assign closed = %d %s, want 409", rec.Code, rec.Body)
	}
	if rec := sendTo(gated, http.MethodPost, "/board-items/"+b.ID+"/close", ""); rec.Code != http.StatusConflict {
		t.Fatalf("second close = %d %s, want 409", rec.Code, rec.Body)
	}
}
