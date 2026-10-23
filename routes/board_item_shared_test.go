package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-shared/geo"
)

func TestASharedBoardItemOpensWithoutSigningInAndNamesNoRaiserAddressOrPin(t *testing.T) {
	_, anon, cm := raisedItemsMount(t)
	ctx := context.Background()
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Streetlights out", Outcome: "Reported to the utility"})
	if err != nil {
		t.Fatal(err)
	}
	pin := geo.NewPoint(36.817223, -1.286389)
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Dark road", Topic: "lighting", Publishable: true, RaisedBy: "amina", Pin: &pin, Address: "12 Kenyatta Avenue", Area: "Central"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, r.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.Endorse(ctx, b.ID, "baraka"); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	anon.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/board-items/"+b.ID+"/shared", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("anonymous open = %d %s, want 200", rec.Code, rec.Body)
	}
	var got catalog.SharedBoardItem
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Title != "Streetlights out" || got.Theme != "lighting" || got.Area != "Central" || got.MeToo != 1 || got.Status != "Reported to the utility" {
		t.Fatalf("shared = %+v", got)
	}
	for _, withheld := range []string{"amina", "Kenyatta", "36.81", "-1.28"} {
		if strings.Contains(rec.Body.String(), withheld) {
			t.Errorf("the shared item carried %q: %s", withheld, rec.Body)
		}
	}
}

func TestAnUnknownSharedBoardItemAnswersTheSameAsOneWithNothingPublishable(t *testing.T) {
	_, anon, cm := raisedItemsMount(t)
	ctx := context.Background()
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Nothing grouped yet"})
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, key := range []string{b.ID, "no-such-item"} {
		rec := httptest.NewRecorder()
		anon.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/board-items/"+key+"/shared", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("open %s = %d %s, want 404", key, rec.Code, rec.Body)
		}
		bodies = append(bodies, rec.Body.String())
	}
	if bodies[0] != bodies[1] {
		t.Fatalf("an empty board item and an unknown key answered differently: %q vs %q", bodies[0], bodies[1])
	}
}
