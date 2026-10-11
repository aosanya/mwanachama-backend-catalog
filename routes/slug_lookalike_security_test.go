package routes_test

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestASlugThatOnlyLooksLikeAPublicEntryIsRefusedAndNeverOpensIt(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	m := mountInto(t, db, filepath.Join("..", "spec", "examples", "agency.catalog.json"))

	if code, body := send(m.operator, http.MethodPut, "/entries/alpha", `{"name":"A","summary":"s","doc":"{\"x\":1}"}`); code != http.StatusOK {
		t.Fatalf("precondition: upsert alpha answered %d %s", code, body)
	}
	if code, body := send(m.operator, http.MethodPost, "/entries/alpha/visibility", `{"visibility":"public"}`); code != http.StatusOK {
		t.Fatalf("precondition: publish alpha answered %d %s", code, body)
	}
	if code, body := send(m.anon, http.MethodGet, "/entries/alpha", ""); code != http.StatusOK {
		t.Fatalf("precondition: anonymous read of alpha answered %d %s", code, body)
	}

	for _, slug := range []string{"ALPHA", "alpha%0A", "alpha%00", "alpha%E2%80%8B", "%D0%B0lpha", "%EF%BD%81lpha", "alph%C3%A0"} {
		if code, body := send(m.operator, http.MethodPut, "/entries/"+slug, `{"name":"X","summary":"s","doc":"{\"x\":1}"}`); code != http.StatusBadRequest {
			t.Errorf("upsert of look-alike slug %q answered %d %s, want 400 so no second entry reads like alpha", slug, code, body)
		}
		if code, body := send(m.anon, http.MethodGet, "/entries/"+slug, ""); code != http.StatusNotFound {
			t.Errorf("anonymous read of look-alike slug %q answered %d %s, want 404 exactly as a missing slug", slug, code, body)
		}
	}

	_, listed := send(m.operator, http.MethodGet, "/entries", "")
	if n := strings.Count(listed, `"slug":`); n != 1 {
		t.Errorf("only alpha should be stored, the listing holds %d entries: %s", n, listed)
	}
}
