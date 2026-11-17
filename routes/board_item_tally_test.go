package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestTheAreaTallyIsServedGatedAndNarrowedByTheQueryThroughTheMux(t *testing.T) {
	gated, anon, cm := raisedItemsMount(t)
	ctx := context.Background()
	first, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Streetlights out"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Market drainage"}); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.Endorse(ctx, first.ID, "chiku"); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	anon.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/board-items/tally-by-area", nil))
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("anonymous tally = %d %s, want it unrouted", rec.Code, rec.Body)
	}

	var all, one []catalog.BoardItemAreaTally
	rec = sendTo(gated, http.MethodGet, "/board-items/tally-by-area", "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &all) != nil || len(all) != 2 {
		t.Fatalf("tally = %d %s, want both board items", rec.Code, rec.Body)
	}
	rec = sendTo(gated, http.MethodGet, "/board-items/tally-by-area?board_item_id="+first.ID, "")
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &one) != nil || len(one) != 1 || one[0].ID != first.ID {
		t.Fatalf("narrowed tally = %d %s, want just the first", rec.Code, rec.Body)
	}
	if rows := one[0].Areas; len(rows) != 1 || rows[0].Area != "" || rows[0].MeToo != 1 || rows[0].Population != nil {
		t.Fatalf("rows = %+v, want one unplaced Me too and a null population", rows)
	}
}
