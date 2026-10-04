package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func issueKey(t *testing.T, h http.Handler, slug string) (key, linkID string) {
	t.Helper()
	rec := do(t, h, http.MethodPost, "/entries/"+slug+"/links", `{"label":"reviewer"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("precondition: issuing a key for %s answered %d %s", slug, rec.Code, rec.Body.String())
	}
	var out struct {
		Key  string `json:"key"`
		Link struct {
			ID string `json:"id"`
		} `json:"link"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.Key == "" || out.Link.ID == "" {
		t.Fatalf("precondition: issue answer carried no key or link id: %s", rec.Body.String())
	}
	return out.Key, out.Link.ID
}

func TestShareKeyOpensOnlyItsOwnEntryInItsExactSpellingWhileLive(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "alpha", catalog.VisibilityPrivate)
	seed(t, cm, ctx, "beta", catalog.VisibilityPrivate)

	key, linkID := issueKey(t, h, "alpha")
	if rec := do(t, h, http.MethodGet, "/entries/alpha?k="+key, ""); rec.Code != http.StatusOK {
		t.Fatalf("precondition: alpha's own key should open alpha, got %d %s", rec.Code, rec.Body.String())
	}
	missing := do(t, h, http.MethodGet, "/entries/no-such-entry", "").Body.String()

	refused := map[string]string{
		"another private entry": "/entries/beta?k=" + key,
		"upper-cased key":       "/entries/alpha?k=" + strings.ToUpper(key),
		"trailing space":        "/entries/alpha?k=" + key + "%20",
		"leading space":         "/entries/alpha?k=%20" + key,
		"trailing NUL":          "/entries/alpha?k=" + key + "%00",
		"truncated key":         "/entries/alpha?k=" + key[:63],
		"upper-cased slug":      "/entries/ALPHA?k=" + key,
		"wrong key bound first": "/entries/alpha?k=wrong&k=" + key,
		"empty key":             "/entries/alpha?k=",
	}
	for name, path := range refused {
		rec := do(t, h, http.MethodGet, path, "")
		if rec.Code != http.StatusNotFound || rec.Body.String() != missing {
			t.Errorf("%s: GET %s answered %d %s; a share key must open only its own entry in its exact spelling, "+
				"refused exactly as a missing slug (%s)", name, path, rec.Code, rec.Body.String(), missing)
		}
	}

	if rec := do(t, h, http.MethodPost, "/links/"+linkID+"/retire", ""); rec.Code != http.StatusOK {
		t.Fatalf("precondition: retiring alpha's link answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, h, http.MethodGet, "/entries/alpha?k="+key, ""); rec.Code != http.StatusNotFound || rec.Body.String() != missing {
		t.Errorf("a retired key still answered %d %s for alpha; it should be refused exactly as a missing slug", rec.Code, rec.Body.String())
	}

	betaKey, _ := issueKey(t, h, "beta")
	if _, err := cm.SetVisibility(ctx, "beta", catalog.VisibilityPublic); err != nil {
		t.Fatalf("precondition: making beta public: %v", err)
	}
	if _, err := cm.SetVisibility(ctx, "beta", catalog.VisibilityPrivate); err != nil {
		t.Fatalf("precondition: making beta private again: %v", err)
	}
	if rec := do(t, h, http.MethodGet, "/entries/beta?k="+betaKey, ""); rec.Code != http.StatusNotFound || rec.Body.String() != missing {
		t.Errorf("a key issued before beta left private and returned still answered %d %s; leaving private must retire it for good",
			rec.Code, rec.Body.String())
	}
}
