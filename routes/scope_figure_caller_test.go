package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAScopeFigureIsRecordedByTheCallerNeverTheBody(t *testing.T) {
	gated, _, _ := callerMount(t, "clerk-1")
	rec := sendTo(gated, http.MethodPost, "/scope-figures", `{"scope_id":"s-1","value":4200,"as_of":"2024-06-30T00:00:00Z","source":"survey 2024","recorded_by":"forged","id":"forged"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("record = %d %s", rec.Code, rec.Body.String())
	}
	rec = sendTo(gated, http.MethodGet, "/scope-figures/s-1", "")
	var got struct {
		ID         string `json:"id"`
		Value      int    `json:"value"`
		RecordedBy string `json:"recorded_by"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("read = %d %s", rec.Code, rec.Body.String())
	}
	if got.RecordedBy != "clerk-1" || got.ID == "forged" || got.Value != 4200 {
		t.Fatalf("figure = %+v, want recorded_by the caller and a minted id", got)
	}
	if rec := sendTo(gated, http.MethodGet, "/scope-figures/s-9", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown scope = %d", rec.Code)
	}
}
