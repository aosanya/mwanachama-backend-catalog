package routes_test

import (
	"net/http"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/routes"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func TestCAT16_PrivateSlugCostsTheStoreTheSameAsAMissingOne(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s, err := catalog.LoadSpec(filepath.Join("..", "spec", "examples", "agency.catalog.json"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	mux := http.NewServeMux()
	for _, r := range routes.Routes(cm) {
		mux.Handle(r.Method+" "+r.Path, r.Handler)
	}

	var statements atomic.Int64
	count := func(*gorm.DB) { statements.Add(1) }
	if err := db.Callback().Query().After("gorm:query").Register("cat16:count_query", count); err != nil {
		t.Fatalf("register query counter: %v", err)
	}
	if err := db.Callback().Row().After("gorm:row").Register("cat16:count_row", count); err != nil {
		t.Fatalf("register row counter: %v", err)
	}
	if err := db.Callback().Raw().After("gorm:raw").Register("cat16:count_raw", count); err != nil {
		t.Fatalf("register raw counter: %v", err)
	}

	seed(t, cm, t.Context(), "private-one", catalog.VisibilityPrivate)
	seed(t, cm, t.Context(), "pending-one", catalog.VisibilityPending)
	if _, _, err := cm.IssueShareLink(t.Context(), "private-one", "for a reviewer", "operator"); err != nil {
		t.Fatalf("precondition: issue a live key on the private entry: %v", err)
	}

	cost := func(path string) (int64, int, string) {
		statements.Store(0)
		rec := do(t, mux, "GET", path, "")
		return statements.Load(), rec.Code, rec.Body.String()
	}

	missing, missingCode, missingBody := cost("/entries/no-such-entry?k=not-a-key")
	pending, pendingCode, pendingBody := cost("/entries/pending-one?k=not-a-key")
	private, privateCode, privateBody := cost("/entries/private-one?k=not-a-key")

	if missingCode != http.StatusNotFound || pendingCode != http.StatusNotFound || privateCode != http.StatusNotFound ||
		missingBody != privateBody || pendingBody != privateBody {
		t.Fatalf("precondition: all three refusals must be the same 404; got missing=%d %s pending=%d %s private=%d %s",
			missingCode, missingBody, pendingCode, pendingBody, privateCode, privateBody)
	}
	if missing == 0 {
		t.Fatalf("precondition: the statement counter saw nothing for the missing slug")
	}

	if private != missing || pending != missing {
		t.Errorf("CAT16 (P1) open hole: an anonymous GET /entries/{slug}?k=<wrong> costs the store %d statement(s) for a private entry, "+
			"%d for a pending one and %d for a slug that does not exist; every refusal should do the same work, "+
			"because the extra key lookup is a ~225µs difference a stranger measures over the socket to list the private catalog",
			private, pending, missing)
	}
}
