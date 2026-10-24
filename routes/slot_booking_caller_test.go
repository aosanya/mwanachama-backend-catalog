package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestABookingIsTheCallersAndAnotherHolderCannotCancelIt(t *testing.T) {
	gated, _, cm := callerMount(t, "amina")
	rec := sendTo(gated, http.MethodPost, "/slots", `{"subject_id":"svc-1","starts_at":"2026-11-02T09:00:00Z","ends_at":"2026-11-02T09:30:00Z","capacity":1}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("offer = %d %s", rec.Code, rec.Body)
	}
	var slot catalog.Slot
	if err := json.Unmarshal(rec.Body.Bytes(), &slot); err != nil {
		t.Fatal(err)
	}
	rec = sendTo(gated, http.MethodPost, "/slots/"+slot.ID+"/bookings", `{"holder_id":"someone-else"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("book = %d %s", rec.Code, rec.Body)
	}
	var b catalog.Booking
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if b.HolderID != "amina" {
		t.Fatalf("holder = %q, want the caller", b.HolderID)
	}
	if rec := sendTo(gated, http.MethodPost, "/slots/"+slot.ID+"/bookings", ``); rec.Code != http.StatusConflict {
		t.Fatalf("second book = %d %s", rec.Code, rec.Body)
	}
	other := callerMountOver(t, cm, "baraka")
	if rec := sendTo(other, http.MethodPost, "/slots/"+slot.ID+"/bookings", ``); rec.Code != http.StatusConflict {
		t.Fatalf("full book = %d %s", rec.Code, rec.Body)
	}
	if rec := sendTo(other, http.MethodPost, "/bookings/"+b.ID+"/cancel", `{"reason":"mine now"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("other cancel = %d %s", rec.Code, rec.Body)
	}
	if rec := sendTo(other, http.MethodPost, "/bookings/"+b.ID+"/staff-cancel", `{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("staff cancel without reason = %d %s", rec.Code, rec.Body)
	}
	rec = sendTo(gated, http.MethodGet, "/bookings/mine", ``)
	var mine []catalog.Booking
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &mine) != nil || len(mine) != 1 {
		t.Fatalf("mine = %d %s", rec.Code, rec.Body)
	}
	rec = sendTo(other, http.MethodPost, "/bookings/"+b.ID+"/staff-cancel", `{"reason":"office closed"}`)
	var cancelled catalog.Booking
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &cancelled) != nil || cancelled.CancelledBy != "baraka" {
		t.Fatalf("staff cancel = %d %s", rec.Code, rec.Body)
	}
	rec = sendTo(gated, http.MethodGet, "/bookings/by-day?day=2026-11-02&subject_id=svc-1", ``)
	var day []catalog.Booking
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &day) != nil || len(day) != 1 {
		t.Fatalf("day = %d %s", rec.Code, rec.Body)
	}
	if rec := sendTo(gated, http.MethodPost, "/bookings/"+b.ID+"/cancel", `{}`); rec.Code != http.StatusConflict {
		t.Fatalf("cancel twice = %d %s", rec.Code, rec.Body)
	}
}
