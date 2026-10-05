package routes_test

import (
	"fmt"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestOversizedSlugOnTheAnonymousDoorsAnswersAsAMissingOneAndStoresNothing(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "open-one", catalog.VisibilityPublic)

	if rec := do(t, h, "GET", "/entries/open-one", ""); rec.Code != 200 {
		t.Fatalf("precondition: the public entry answered %d %s, want 200", rec.Code, rec.Body.String())
	}
	missingOpen := do(t, h, "GET", "/entries/never-filed", "")
	missingView := do(t, h, "POST", "/page-views", `{"page":"entry","slug":"never-filed"}`)
	if missingOpen.Code != 404 || missingView.Code != 404 {
		t.Fatalf("precondition: a never-filed slug answered open %d, view %d; want 404 for both", missingOpen.Code, missingView.Code)
	}

	for _, n := range []int{1000, 100000, 900000} {
		slug := strings.Repeat("a", n)
		open := do(t, h, "GET", "/entries/"+slug, "")
		if open.Code != missingOpen.Code || open.Body.String() != missingOpen.Body.String() {
			t.Errorf("GET /entries/<%d-byte slug> answered %d %q; a never-filed slug answers %d %q, and the two must not differ", n, open.Code, open.Body.String(), missingOpen.Code, missingOpen.Body.String())
		}
		view := do(t, h, "POST", "/page-views", fmt.Sprintf(`{"page":"entry","slug":%q}`, slug))
		if view.Code != missingView.Code || view.Body.String() != missingView.Body.String() {
			t.Errorf("POST /page-views entry <%d-byte slug> answered %d %q; a never-filed slug answers %d %q", n, view.Code, view.Body.String(), missingView.Code, missingView.Body.String())
		}
		do(t, h, "POST", "/page-views", fmt.Sprintf(`{"page":"index","slug":%q}`, slug))
	}

	views, err := cm.ListPageViews(ctx)
	if err != nil {
		t.Fatalf("ListPageViews: %v", err)
	}
	for _, v := range views {
		if v.Slug != "" {
			t.Errorf("an anonymous page view stored a %d-byte slug on page %q; only the index counter may exist, with no slug", len(v.Slug), v.Page)
		}
	}
	if len(views) != 1 || views[0].Page != "index" {
		t.Errorf("oversized anonymous page views left %d counter rows %+v, want exactly the one index counter", len(views), views)
	}
}
