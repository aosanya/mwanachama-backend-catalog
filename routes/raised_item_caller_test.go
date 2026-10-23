package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/routes"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func callerMount(t *testing.T, caller string) (http.Handler, http.Handler, catalog.CatalogManager) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	s, err := catalog.LoadSpec(filepath.Join("..", "spec", "examples", "agency.catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatal(err)
	}
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	gated, anon := http.NewServeMux(), http.NewServeMux()
	for _, r := range routes.RoutesFor(cm, routes.Mount{Caller: func(context.Context) string { return caller }}) {
		gated.Handle(r.Method+" "+r.Path, r.Handler)
	}
	for _, r := range routes.PublicRoutes(cm) {
		anon.Handle(r.Method+" "+r.Path, r.Handler)
	}
	return gated, anon, cm
}

func sendTo(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestARaisedItemIsTheCallersWhateverTheBodyClaims(t *testing.T) {
	gated, _, _ := callerMount(t, "amina")
	rec := sendTo(gated, http.MethodPost, "/raised-items", `{"title":"Dark road","raised_by":"someone-else"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("raise = %d %s", rec.Code, rec.Body)
	}
	var r catalog.RaisedItem
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	if r.RaisedBy != "amina" || r.FiledBy != "" {
		t.Fatalf("raised_by %q filed_by %q, want amina and nobody", r.RaisedBy, r.FiledBy)
	}
	rec = sendTo(gated, http.MethodGet, "/raised-items/mine", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Dark road") {
		t.Fatalf("mine = %d %s", rec.Code, rec.Body)
	}
}

func TestAFiledItemNamesItsRaiserFromTheBodyAndItsFilerFromTheCaller(t *testing.T) {
	gated, _, _ := callerMount(t, "clerk-1")
	rec := sendTo(gated, http.MethodPost, "/raised-items/filed", `{"title":"Burst pipe","raised_by":"register-7","channel":"phone","filed_by":"forged"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("file = %d %s", rec.Code, rec.Body)
	}
	var r catalog.RaisedItem
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	if r.RaisedBy != "register-7" || r.FiledBy != "clerk-1" {
		t.Fatalf("raised_by %q filed_by %q, want register-7 and clerk-1", r.RaisedBy, r.FiledBy)
	}
}

func TestBoardMeTooAndRepliesAnswerThroughTheDeclaredTableAndNeverAnonymously(t *testing.T) {
	gated, anon, _ := callerMount(t, "amina")
	var b catalog.BoardItem
	rec := sendTo(gated, http.MethodPost, "/board-items", `{"title":"Streetlights out"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("put on board = %d %s", rec.Code, rec.Body)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &b)
	var r catalog.RaisedItem
	_ = json.Unmarshal(sendTo(gated, http.MethodPost, "/raised-items", `{"title":"Dark road","publishable":true}`).Body.Bytes(), &r)
	if rec := sendTo(gated, http.MethodPut, "/raised-items/"+r.ID+"/board-item", `{"board_item_id":"`+b.ID+`"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("group = %d %s", rec.Code, rec.Body)
	}
	if rec := sendTo(gated, http.MethodPut, "/board-items/"+b.ID+"/outcome", `{"outcome":"Fixed"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("outcome = %d %s", rec.Code, rec.Body)
	}
	if rec := sendTo(gated, http.MethodPost, "/endorsements/"+b.ID, ""); rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "1" {
		t.Fatalf("me too = %d %s", rec.Code, rec.Body)
	}
	if rec := sendTo(gated, http.MethodPost, "/endorsements/"+b.ID, ""); strings.TrimSpace(rec.Body.String()) != "1" {
		t.Fatalf("second me too = %d %s, want still 1", rec.Code, rec.Body)
	}
	rec = sendTo(gated, http.MethodPost, "/raised-items/"+r.ID+"/replies", `{"body":"On it","replied_by":"forged"}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"replied_by":"amina"`) {
		t.Fatalf("reply = %d %s", rec.Code, rec.Body)
	}
	if rec := sendTo(gated, http.MethodGet, "/raised-items/"+r.ID+"/replies", ""); !strings.Contains(rec.Body.String(), "On it") {
		t.Fatalf("replies = %d %s", rec.Code, rec.Body)
	}
	if rec := sendTo(gated, http.MethodDelete, "/raised-items/"+r.ID+"/board-item", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("ungroup = %d %s", rec.Code, rec.Body)
	}
	for _, p := range [][2]string{{http.MethodPost, "/raised-items"}, {http.MethodPost, "/board-items"}, {http.MethodGet, "/raised-items/" + r.ID + "/replies"}, {http.MethodPost, "/endorsements/" + b.ID}} {
		if rec := sendTo(anon, p[0], p[1], `{}`); rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("anonymous %s %s = %d, want unrouted", p[0], p[1], rec.Code)
		}
	}
}
