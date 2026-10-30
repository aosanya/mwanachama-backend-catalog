package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestDecideSuggestionOverTheRoutesTakesZeroAndNegativePrioritiesAndKeepsAnOmittedOne(t *testing.T) {
	op, _, _ := municipalityWard(t)
	id := fileArrival(t, op, "priority-probe", "counter")

	priorityAfter := func(body string) float64 {
		t.Helper()
		rec := do(t, op, "POST", "/suggestions/"+id+"/decide", body)
		if rec.Code != http.StatusOK {
			t.Fatalf("POST /decide %s answered %d %s", body, rec.Code, rec.Body.String())
		}
		var got struct {
			Priority float64 `json:"priority"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode %s: %v", rec.Body.String(), err)
		}
		return got.Priority
	}

	if got := priorityAfter(`{"status":"approved","priority":0}`); got != 0 {
		t.Errorf("priority 0 left %v, want 0", got)
	}
	if got := priorityAfter(`{"status":"approved","priority":-10}`); got != -10 {
		t.Errorf("priority -10 left %v, want -10", got)
	}
	if got := priorityAfter(`{"status":"proposed"}`); got != -10 {
		t.Errorf("a decide naming no priority left %v, want the stored -10", got)
	}
}
