package routes_test

import (
	"context"
	"io"
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

type mountUnderTest struct {
	cm       catalog.CatalogManager
	operator http.Handler
	anon     http.Handler
}

func mountInto(t *testing.T, db *gorm.DB, path string) mountUnderTest {
	t.Helper()
	s, err := catalog.LoadSpec(path)
	if err != nil {
		t.Fatalf("load %s: %v", path, err)
	}
	s.Instance = "wakala"
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate %s: %v", path, err)
	}
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("manager %s: %v", path, err)
	}
	operator := http.NewServeMux()
	for _, r := range routes.Routes(cm) {
		operator.Handle(r.Method+" "+r.Path, r.Handler)
	}
	anon := http.NewServeMux()
	for _, r := range routes.PublicRoutesFor(cm, routes.Mount{Anonymous: municipalityAnonymous}) {
		anon.Handle(r.Method+" "+r.Path, r.Handler)
	}
	return mountUnderTest{cm: cm, operator: operator, anon: anon}
}

func send(h http.Handler, method, path, body string) (int, string) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	b, _ := io.ReadAll(rec.Result().Body)
	return rec.Code, string(b)
}

func TestTwoMountsOfOneInstanceNeverReachEachOthersRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	storefront := mountInto(t, db, filepath.Join("..", "spec", "examples", "agency.catalog.json"))
	board := mountInto(t, db, filepath.Join("testdata", municipalityReportsSpec))
	ctx := context.Background()

	for _, step := range [][3]string{
		{http.MethodPut, "/entries/alpha", `{"name":"A","summary":"s","doc":"{\"x\":1}"}`},
		{http.MethodPost, "/entries/alpha/visibility", `{"visibility":"public"}`},
		{http.MethodPut, "/entries/beta", `{"name":"B","summary":"s","doc":"{\"x\":1}"}`},
		{http.MethodPost, "/entries/beta/visibility", `{"visibility":"private"}`},
		{http.MethodPut, "/settings/k1", `{"value":"storefront-only"}`},
	} {
		if code, body := send(storefront.operator, step[0], step[1], step[2]); code != http.StatusOK {
			t.Fatalf("precondition: storefront %s %s answered %d %s", step[0], step[1], code, body)
		}
	}
	if code, body := send(storefront.anon, http.MethodPost, "/page-views", `{"page":"entry","slug":"alpha"}`); code != http.StatusNoContent {
		t.Fatalf("precondition: an anonymous storefront page view answered %d %s", code, body)
	}
	link, key, err := storefront.cm.IssueShareLink(ctx, "beta", "l", "op")
	if err != nil {
		t.Fatalf("precondition: issuing a storefront share key: %v", err)
	}
	if code, _ := send(storefront.anon, http.MethodGet, "/entries/beta?k="+key, ""); code != http.StatusOK {
		t.Fatalf("precondition: the storefront key does not open its own private entry (%d)", code)
	}
	if code, body := send(board.anon, http.MethodPost, "/suggestions", `{"slug":"resident-1","title":"t","why":"w","proposal":"{\"a\":1}"}`); code != http.StatusCreated {
		t.Fatalf("precondition: an anonymous report on the board answered %d %s", code, body)
	}
	filed, err := board.cm.ListSuggestions(ctx)
	if err != nil || len(filed) != 1 {
		t.Fatalf("precondition: the board holds %d reports (%v); want 1", len(filed), err)
	}

	for _, probe := range []struct {
		h            http.Handler
		method, path string
		body         string
		want         int
	}{
		{board.anon, http.MethodGet, "/entries/alpha", "", http.StatusNotFound},
		{board.anon, http.MethodGet, "/entries/beta?k=" + key, "", http.StatusNotFound},
		{board.anon, http.MethodPost, "/page-views", `{"page":"entry","slug":"alpha"}`, http.StatusNotFound},
		{board.operator, http.MethodGet, "/settings/k1", "", http.StatusNotFound},
		{board.operator, http.MethodPost, "/links/" + link.ID + "/retire", "", http.StatusNotFound},
		{board.operator, http.MethodDelete, "/entries/alpha", "", http.StatusNotFound},
		{storefront.operator, http.MethodGet, "/suggestions/" + filed[0].ID, "", http.StatusNotFound},
		{storefront.operator, http.MethodPost, "/suggestions/" + filed[0].ID + "/decide", `{"status":"rejected"}`, http.StatusNotFound},
	} {
		if code, body := send(probe.h, probe.method, probe.path, probe.body); code != probe.want {
			t.Errorf("a %s %s across mounts answered %d %s; want %d — two mounts of one instance must never reach each other's rows", probe.method, probe.path, code, body, probe.want)
		}
	}

	for _, listing := range []struct {
		h       http.Handler
		path    string
		foreign string
	}{
		{board.anon, "/public/entries", "alpha"},
		{storefront.operator, "/suggestions", "resident-1"},
		{board.operator, "/page-views", "alpha"},
	} {
		if _, body := send(listing.h, http.MethodGet, listing.path, ""); strings.Contains(body, listing.foreign) {
			t.Errorf("GET %s carried the other mount's %q: %s", listing.path, listing.foreign, body)
		}
	}

	if code, _ := send(storefront.anon, http.MethodGet, "/entries/beta?k="+key, ""); code != http.StatusOK {
		t.Errorf("the storefront key stopped opening its entry after the board was asked to retire it (%d)", code)
	}
	if code, _ := send(storefront.anon, http.MethodGet, "/entries/alpha", ""); code != http.StatusOK {
		t.Errorf("the storefront entry stopped opening after the board was asked to delete it (%d)", code)
	}
}
