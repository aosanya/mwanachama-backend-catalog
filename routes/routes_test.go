package routes_test

import (
	"context"
	"encoding/json"
	"errors"
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

const doc = `{"agency":{"name":"Test"},"_template":{"sector":"health"},"modules":[{"code":"auth","enabled":true}]}`

func newServer(t *testing.T) (http.Handler, catalog.CatalogManager, context.Context) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
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
	return mux, cm, context.Background()
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func seed(t *testing.T, cm catalog.CatalogManager, ctx context.Context, slug string, v catalog.Visibility) {
	t.Helper()
	if _, err := cm.CreateEntry(ctx, catalog.Entry{Slug: slug, Name: "Agency", Doc: doc}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if v != catalog.VisibilityPending {
		if _, err := cm.SetVisibility(ctx, slug, v); err != nil {
			t.Fatalf("visibility: %v", err)
		}
	}
}

// Over HTTP, a pending agency, a missing slug and a wrong key must all be
// the same 404 with the same body. A different status or message would hand
// back exactly what the store refuses to tell a caller.
func TestOpenEntry_RefusalsAreIdenticalOverHTTP(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "pending-one", catalog.VisibilityPending)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	var first string
	for _, path := range []string{
		"/entries/no-such-agency",
		"/entries/pending-one",
		"/entries/private-one",
		"/entries/private-one?k=wrong",
	} {
		rec := do(t, h, http.MethodGet, path, "")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s: status = %d, want 404", path, rec.Code)
		}
		if first == "" {
			first = rec.Body.String()
		} else if rec.Body.String() != first {
			t.Errorf("%s: body = %q, want the same refusal as the others (%q)", path, rec.Body.String(), first)
		}
	}
}

func TestOpenEntry_PrivateOpensWithItsKey(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	rec := do(t, h, http.MethodPost, "/entries/private-one/links", `{"label":"pilot","by":"owner"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("issue: status = %d, body %s", rec.Code, rec.Body.String())
	}
	var issued struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &issued); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if issued.Key == "" {
		t.Fatal("no raw key returned")
	}

	ok := do(t, h, http.MethodGet, "/entries/private-one?k="+issued.Key, "")
	if ok.Code != http.StatusOK {
		t.Fatalf("open with key: status = %d, body %s", ok.Code, ok.Body.String())
	}
}

// The raw key is returned once, at issue. Listing links must never carry
// one, nor the stored hash.
func TestListShareLinks_NeverCarriesAKey(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	issue := do(t, h, http.MethodPost, "/entries/private-one/links", `{"label":"pilot"}`)
	if issue.Code != http.StatusCreated {
		t.Fatalf("issue: %d", issue.Code)
	}
	var issued struct {
		Key string `json:"key"`
	}
	_ = json.Unmarshal(issue.Body.Bytes(), &issued)

	list := do(t, h, http.MethodGet, "/entries/private-one/links", "")
	if list.Code != http.StatusOK {
		t.Fatalf("list: %d", list.Code)
	}
	body := list.Body.String()
	if strings.Contains(body, issued.Key) {
		t.Error("the raw key appears in the link listing")
	}
	if strings.Contains(body, "key_hash") || strings.Contains(body, "KeyHash") {
		t.Error("the stored hash appears in the link listing")
	}
}

// The public listing is cards. A document reaching it would put 34 KB per
// row on every index page, and would carry pending documents if the filter
// were ever wrong.
func TestListEntries_CarriesNoDocument(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "public-one", catalog.VisibilityPublic)

	rec := do(t, h, http.MethodGet, "/entries?visibility=public", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), `"modules"`) {
		t.Error("the document reached the listing")
	}
}

func TestRecordPageView_RefusesAnInventedSlug(t *testing.T) {
	h, _, _ := newServer(t)

	rec := do(t, h, http.MethodPost, "/page-views", `{"page":"entry","slug":"invented-one"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestSuggestionRoutes_FileDecideAndFulfil(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "the-answer", catalog.VisibilityPending)

	filed := do(t, h, http.MethodPost, "/suggestions",
		`{"slug":"new-one","title":"Build it","source":"run"}`)
	if filed.Code != http.StatusCreated {
		t.Fatalf("file: status = %d, body %s", filed.Code, filed.Body.String())
	}
	var s catalog.Suggestion
	if err := json.Unmarshal(filed.Body.Bytes(), &s); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// A second filing of the same slug is a conflict, not a second row.
	again := do(t, h, http.MethodPost, "/suggestions",
		`{"slug":"new-one","title":"Build it again","source":"run"}`)
	if again.Code != http.StatusConflict {
		t.Errorf("duplicate: status = %d, want 409", again.Code)
	}

	// Deciding it "built" is refused — only fulfilling may do that.
	bad := do(t, h, http.MethodPost, "/suggestions/"+s.ID+"/decide", `{"status":"built"}`)
	if bad.Code != http.StatusConflict {
		t.Errorf("decide built: status = %d, want 409", bad.Code)
	}

	if rec := do(t, h, http.MethodPost, "/suggestions/"+s.ID+"/decide", `{"status":"approved"}`); rec.Code != http.StatusOK {
		t.Fatalf("approve: status = %d, body %s", rec.Code, rec.Body.String())
	}
	done := do(t, h, http.MethodPost, "/suggestions/"+s.ID+"/fulfil", `{"slug":"the-answer","run_id":"r1"}`)
	if done.Code != http.StatusOK {
		t.Fatalf("fulfil: status = %d, body %s", done.Code, done.Body.String())
	}
	var built catalog.Suggestion
	_ = json.Unmarshal(done.Body.Bytes(), &built)
	if built.Status != catalog.SuggestionStatusBuilt {
		t.Errorf("Status = %q, want built", built.Status)
	}
}

func TestRoutes_BadJSONIsABadRequest(t *testing.T) {
	h, _, _ := newServer(t)
	rec := do(t, h, http.MethodPost, "/suggestions", `{"slug":`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestOperatorRoutesWith_GatesOnlyTheGatedHalf(t *testing.T) {
	_, cm, _ := newServer(t)

	var asked []string
	authorize := func(ctx context.Context, action string) error {
		asked = append(asked, action)
		return errors.New("nope")
	}

	public := routes.PublicRoutes(cm)
	gated := routes.OperatorRoutesWith(cm, authorize)

	if len(public) != len(routes.AnonymousActions) {
		t.Fatalf("public routes = %d, want %d", len(public), len(routes.AnonymousActions))
	}
	if len(gated) == 0 {
		t.Fatal("no gated routes were built")
	}

	for _, r := range public {
		rec := httptest.NewRecorder()
		r.Handler(rec, httptest.NewRequest(r.Method, "/entries/x", nil))
		if rec.Code == http.StatusForbidden {
			t.Errorf("a storefront route %s was gated; it never had a caller to authorize", r.Action)
		}
	}
	if len(asked) != 0 {
		t.Errorf("the authorizer was asked about a public action: %v", asked)
	}

	for _, r := range gated {
		rec := httptest.NewRecorder()
		r.Handler(rec, httptest.NewRequest(r.Method, "/entries", nil))
		if rec.Code != http.StatusForbidden {
			t.Errorf("gated route %s answered %d, want 403", r.Action, rec.Code)
		}
	}
	if len(asked) != len(gated) {
		t.Errorf("the authorizer was asked %d times for %d gated routes", len(asked), len(gated))
	}
}
