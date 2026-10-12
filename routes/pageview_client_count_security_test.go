package routes_test

import (
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestPageView_AnonymousBodyNeverSetsTheCount(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "open-one", catalog.VisibilityPublic)

	for _, body := range []string{
		`{"page":"entry","slug":"open-one","views":1000000}`,
		`{"page":"entry","slug":"open-one","Views":1000000}`,
		`{"page":"entry","slug":"open-one","VIEWS":-1000000}`,
		`{"page":"index","views":1000000}`,
		`{"page":"index","views":-5}`,
	} {
		if got := do(t, h, "POST", "/page-views", body); got.Code != 204 {
			t.Fatalf("precondition: an anonymous POST /page-views %s answered %d %q, want 204", body, got.Code, got.Body.String())
		}
	}

	views, err := cm.ListPageViews(ctx)
	if err != nil {
		t.Fatalf("ListPageViews: %v", err)
	}
	want := map[string]int{"entry/open-one": 3, "index/": 2}
	if len(views) != len(want) {
		t.Fatalf("five anonymous page views left counters %+v, want exactly %v", views, want)
	}
	for _, v := range views {
		key := string(v.Page) + "/" + v.Slug
		if int(v.Views) != want[key] {
			t.Errorf("counter %s reads %d after a body claiming its count, want %d: an anonymous caller must only ever add one", key, v.Views, want[key])
		}
	}
}
