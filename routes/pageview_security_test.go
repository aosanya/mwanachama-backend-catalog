package routes_test

import (
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT11_PageViewAnswersAnUnpublishedSlugLikeAMissingOne(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "pending-one", catalog.VisibilityPending)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)
	for _, slug := range []string{"pending-one", "private-one", "never-filed"} {
		if rec := do(t, h, "GET", "/entries/"+slug, ""); rec.Code != http.StatusNotFound {
			t.Fatalf("precondition: an anonymous open of %q must be 404, got %d %s", slug, rec.Code, rec.Body.String())
		}
	}

	missing := do(t, h, "POST", "/page-views", `{"page":"entry","slug":"never-filed"}`)
	for _, slug := range []string{"pending-one", "private-one"} {
		got := do(t, h, "POST", "/page-views", `{"page":"entry","slug":"`+slug+`"}`)
		if got.Code != missing.Code || got.Body.String() != missing.Body.String() {
			t.Errorf("CAT11 (P1) open hole: an anonymous POST /page-views for the unpublished slug %q answered %d %q, "+
				"but for a slug that was never filed it answered %d %q; the two must be indistinguishable, "+
				"or the page-view door enumerates the unpublished catalog that GET /entries/{slug} refuses to",
				slug, got.Code, got.Body.String(), missing.Code, missing.Body.String())
		}
	}
}
