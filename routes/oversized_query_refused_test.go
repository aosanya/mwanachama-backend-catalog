package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/routes"
)

const queryParameterCap = 10000

func floodQuery(key string, n int) string {
	return strings.Repeat(key+"=x&", n)
}

func TestAQueryTheServerCannotParseIsRefusedWith400NotReadAsUnfiltered(t *testing.T) {
	if _, err := url.ParseQuery(floodQuery("status", queryParameterCap+1)); err == nil {
		t.Skip("precondition: this Go toolchain parses a query of more than 10000 parameters")
	}
	h, _, _ := newServer(t)

	for _, path := range []string{
		"/suggestions?" + floodQuery("status", queryParameterCap+1),
		"/suggestions?status=proposed&" + floodQuery("noise", queryParameterCap),
		"/entries?" + floodQuery("limit", queryParameterCap+1),
	} {
		rec := do(t, h, "GET", path, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("GET %.40s... answered %d %.120s; want 400", path, rec.Code, rec.Body.String())
			continue
		}
		body := rec.Body.String()
		if !strings.Contains(body, "query") {
			t.Errorf("GET %.40s... 400 body %q does not say the query was refused", path, body)
		}
		if w := namesTheDriver(body); w != "" {
			t.Errorf("GET %.40s... 400 body names %q: %s", path, w, body)
		}
	}
}

func TestAQueryAtTheCapIsStillAnswered(t *testing.T) {
	h, _, _ := newServer(t)
	rec := do(t, h, "GET", "/suggestions?"+floodQuery("status", queryParameterCap-1), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /suggestions with %d statuses answered %d %s; want 200", queryParameterCap-1, rec.Code, rec.Body.String())
	}
}

func TestTheAnonymousDoorRefusesAnUnparseableQueryToo(t *testing.T) {
	_, cm, _ := newServer(t)
	mux := http.NewServeMux()
	for _, r := range routes.PublicRoutes(cm) {
		mux.Handle(r.Method+" "+r.Path, r.Handler)
	}
	req := httptest.NewRequest("GET", "/entries/anything?"+floodQuery("key", queryParameterCap+1), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("anonymous GET /entries/anything with an unparseable query answered %d %s; want 400", rec.Code, rec.Body.String())
	}
}

func TestOnlyTheParameterCapIsRefusedACapErrorTheGuardRecognises(t *testing.T) {
	_, capErr := url.ParseQuery(floodQuery("status", queryParameterCap+1))
	if capErr == nil {
		t.Skip("precondition: this Go toolchain parses a query of more than 10000 parameters")
	}
	h, _, _ := newServer(t)
	rec := do(t, h, "GET", "/suggestions?"+floodQuery("status", queryParameterCap+1), "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("an over-cap query answered %d %s; want 400", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "too many parameters") {
		t.Errorf("an over-cap query answered %s; want the cap named", rec.Body.String())
	}
	if !routes.IsQueryCapError(capErr) {
		t.Errorf("the guard does not recognise the toolchain's cap error %q", capErr)
	}
	for _, other := range []string{"status=%zz", "a=1;b=2", "status=proposed;x=1"} {
		_, err := url.ParseQuery(other)
		if err == nil {
			t.Fatalf("precondition: %q parses without an error", other)
		}
		if routes.IsQueryCapError(err) {
			t.Errorf("the guard takes the error of %q for the cap: %v", other, err)
		}
	}
}

func TestABadEscapeOrASemicolonDropsItsPairAndTheRestOfTheQueryIsUsed(t *testing.T) {
	h, cm, ctx := newServer(t)
	for _, slug := range []string{"alpha", "beta", "gamma"} {
		seed(t, cm, ctx, slug, catalog.VisibilityPublic)
	}

	count := func(path string) int {
		t.Helper()
		rec := do(t, h, "GET", path, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s answered %d %s; want 200", path, rec.Code, rec.Body.String())
		}
		var rows []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
			t.Fatalf("GET %s answered a body that is not a list: %v", path, err)
		}
		return len(rows)
	}

	if all := count("/entries"); all != 3 {
		t.Fatalf("precondition: /entries lists %d rows, want 3", all)
	}
	for _, path := range []string{
		"/entries?limit=1&bad=%zz",
		"/entries?limit=1&a=1;b=2",
		"/entries?limit=1&%zz=1",
		"/entries?limit=1&status=proposed;x=1",
	} {
		if got := count(path); got != 1 {
			t.Errorf("GET %s listed %d rows; want 1, the limit read and only the bad pair dropped", path, got)
		}
	}
	for _, path := range []string{"/suggestions?status=%zz", "/suggestions?a=1;b=2", "/suggestions?status=proposed;x=1"} {
		if rec := do(t, h, "GET", path, ""); rec.Code != http.StatusOK {
			t.Errorf("GET %s answered %d %s; want 200", path, rec.Code, rec.Body.String())
		}
	}
}

func TestTheAnonymousDoorReadsAQueryWithABadEscapeOrASemicolon(t *testing.T) {
	_, cm, _ := newServer(t)
	mux := http.NewServeMux()
	for _, r := range routes.PublicRoutes(cm) {
		mux.Handle(r.Method+" "+r.Path, r.Handler)
	}
	for _, query := range []string{"key=%zz", "a=1;b=2"} {
		req := httptest.NewRequest("GET", "/entries/anything?"+query, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code == http.StatusBadRequest && strings.Contains(rec.Body.String(), "query string") {
			t.Errorf("anonymous GET /entries/anything?%s was refused as an unreadable query: %s", query, rec.Body.String())
		}
	}
}
