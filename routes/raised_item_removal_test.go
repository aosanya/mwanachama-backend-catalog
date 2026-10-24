package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/routes"
)

func callerMountOver(t *testing.T, cm catalog.CatalogManager, caller string) http.Handler {
	t.Helper()
	gated := http.NewServeMux()
	for _, r := range routes.RoutesFor(cm, routes.Mount{Caller: func(context.Context) string { return caller }}) {
		gated.Handle(r.Method+" "+r.Path, r.Handler)
	}
	return gated
}

func TestARemovalNamesItsGroundAndRemoverAndReachesOnlyItsRaiser(t *testing.T) {
	gated, _, cm := callerMount(t, "staff-1")
	amina, err := cm.RaiseItem(t.Context(), catalog.RaisedItem{Title: "Dark road", RaisedBy: "amina", Publishable: true})
	if err != nil {
		t.Fatal(err)
	}
	other, err := cm.RaiseItem(t.Context(), catalog.RaisedItem{Title: "Burst pipe", RaisedBy: "baraka", Publishable: true})
	if err != nil {
		t.Fatal(err)
	}
	rec := sendTo(gated, http.MethodPost, "/raised-items/"+amina.ID+"/removal", `{"ground":"Duplicate of an open report","removed_by":"forged","raised_item_id":"`+other.ID+`","id":"chosen","created_at":"2000-01-01T00:00:00Z"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("remove = %d %s", rec.Code, rec.Body)
	}
	var r catalog.Removal
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	if r.RemovedBy != "staff-1" || r.RaisedItemID != amina.ID || r.ID == "chosen" || r.CreatedAt == "2000-01-01T00:00:00Z" {
		t.Fatalf("removal %+v, want staff-1 removing %s with a store id and time", r, amina.ID)
	}
	if rec := sendTo(gated, http.MethodPost, "/raised-items/"+amina.ID+"/removal", `{"ground":"Again"}`); rec.Code != http.StatusConflict {
		t.Fatalf("second removal = %d %s, want 409", rec.Code, rec.Body)
	}
	if rec := sendTo(gated, http.MethodPost, "/raised-items/"+other.ID+"/removal", `{"ground":"   "}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("blank ground = %d %s, want 400", rec.Code, rec.Body)
	}
	if rec := sendTo(gated, http.MethodPost, "/raised-items/no-such-item/removal", `{"ground":"Spam"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown item = %d %s, want 404", rec.Code, rec.Body)
	}

	raiser := callerMountOver(t, cm, "amina")
	rec = sendTo(raiser, http.MethodGet, "/raised-items/mine/removals", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Duplicate of an open report") || !strings.Contains(rec.Body.String(), `"removed_by":"staff-1"`) {
		t.Fatalf("amina's removals = %d %s", rec.Code, rec.Body)
	}
	bystander := callerMountOver(t, cm, "baraka")
	if rec := sendTo(bystander, http.MethodGet, "/raised-items/mine/removals", ""); rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "Duplicate") {
		t.Fatalf("baraka's removals = %d %s, want none of amina's", rec.Code, rec.Body)
	}
}

func TestRemovalAddressesAreNeverAnonymous(t *testing.T) {
	_, anon, cm := callerMount(t, "staff-1")
	r, err := cm.RaiseItem(t.Context(), catalog.RaisedItem{Title: "Dark road", RaisedBy: "amina", Publishable: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range [][2]string{{http.MethodPost, "/raised-items/" + r.ID + "/removal"}, {http.MethodGet, "/raised-items/mine/removals"}} {
		if rec := sendTo(anon, p[0], p[1], `{"ground":"Spam"}`); rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("anonymous %s %s = %d, want unrouted", p[0], p[1], rec.Code)
		}
	}
	for _, a := range routes.AnonymousActions {
		if a == "catalog.raised_item.remove" || a == "catalog.removal.list_mine" {
			t.Fatalf("%s is anonymous", a)
		}
	}
}
