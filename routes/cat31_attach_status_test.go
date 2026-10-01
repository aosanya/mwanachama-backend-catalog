package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCAT31_OpenBug_RejectedArrivalStillCountsAsAttached(t *testing.T) {
	op, _, _ := municipalityWard(t)

	liveID := fileArrival(t, op, "junction-report-0", "counter")
	rejectedID := fileArrival(t, op, "junction-report-1", "phone")

	if rec := do(t, op, "PUT", "/entries/ngong-road-junction", boardItemBody(1)); rec.Code != http.StatusOK {
		t.Fatalf("precondition: opening the board item must be 200, got %d %s", rec.Code, rec.Body.String())
	}

	if rec := do(t, op, "POST", "/suggestions/"+rejectedID+"/decide", `{"status":"rejected"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: rejecting must be 200, got %d %s", rec.Code, rec.Body.String())
	}

	if rec := do(t, op, "POST", "/suggestions/"+liveID+"/matches", `{"slug":"ngong-road-junction","note":"same junction"}`); rec.Code != http.StatusCreated {
		t.Fatalf("precondition: attaching the live arrival must be 201, got %d %s", rec.Code, rec.Body.String())
	}

	attached := do(t, op, "POST", "/suggestions/"+rejectedID+"/matches", `{"slug":"ngong-road-junction","note":"late"}`)
	if attached.Code == http.StatusCreated {
		listed := do(t, op, "GET", "/entries/ngong-road-junction/matches", "")
		var rows []struct {
			SuggestionID string `json:"suggestion_id"`
		}
		if err := json.Unmarshal(listed.Body.Bytes(), &rows); err != nil {
			t.Fatalf("decode matches: %v (%s)", err, listed.Body.String())
		}
		for _, r := range rows {
			if r.SuggestionID == rejectedID {
				t.Fatalf("CAT31 open bug: a suggestion rejected before it was ever attached (%s) was accepted "+
					"by POST /suggestions/{id}/matches (201) and now counts toward ngong-road-junction's "+
					"attached arrivals: %s — a council prioritising this item is counting a report the "+
					"moderator already threw out", rejectedID, listed.Body.String())
			}
		}
	}

	if rec := do(t, op, "POST", "/suggestions/"+liveID+"/decide", `{"status":"rejected"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: rejecting the previously-attached arrival must be 200, got %d %s", rec.Code, rec.Body.String())
	}

	listedAfter := do(t, op, "GET", "/entries/ngong-road-junction/matches", "")
	if listedAfter.Code != http.StatusOK {
		t.Fatalf("precondition: listing matches must be 200, got %d %s", listedAfter.Code, listedAfter.Body.String())
	}
	var afterRows []struct {
		SuggestionID string `json:"suggestion_id"`
	}
	if err := json.Unmarshal(listedAfter.Body.Bytes(), &afterRows); err != nil {
		t.Fatalf("decode matches: %v (%s)", err, listedAfter.Body.String())
	}
	for _, r := range afterRows {
		if r.SuggestionID == liveID {
			t.Fatalf("CAT31 open bug: %s was attached while proposed, then rejected by a reviewer, and still "+
				"lists as attached to ngong-road-junction: %s — rejecting an arrival must drop it from the "+
				"count a council acts on, not leave a stale attachment behind", liveID, listedAfter.Body.String())
		}
	}
}
