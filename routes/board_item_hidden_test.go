package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestHidingABoardItemThroughTheMuxEndsItsShareLinkAndIsNeverAnonymous(t *testing.T) {
	gated, anon, cm := raisedItemsMount(t)
	ctx := context.Background()
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Streetlights out"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Dark road", Publishable: true, RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, r.ID, b.ID); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	anon.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/board-items/"+b.ID+"/hidden", strings.NewReader(`{"hidden":true}`)))
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("anonymous hide = %d %s, want it unrouted", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	gated.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/board-items/"+b.ID+"/hidden", strings.NewReader(`{"hidden":true}`)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("hide = %d %s, want 204", rec.Code, rec.Body)
	}
	rec = httptest.NewRecorder()
	anon.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/board-items/"+b.ID+"/shared", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("hidden share link = %d %s, want 404", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	gated.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/board-items/"+b.ID+"/hidden", strings.NewReader(`{"hidden":false}`)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("show = %d %s, want 204", rec.Code, rec.Body)
	}
	rec = httptest.NewRecorder()
	anon.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/board-items/"+b.ID+"/shared", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("shown share link = %d %s, want 200", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	gated.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/board-items/nope/hidden", strings.NewReader(`{"hidden":true}`)))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown = %d %s, want 404", rec.Code, rec.Body)
	}
}
