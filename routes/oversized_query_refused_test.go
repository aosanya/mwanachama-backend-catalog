package routes_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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
