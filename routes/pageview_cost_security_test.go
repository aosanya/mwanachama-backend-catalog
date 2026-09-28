package routes_test

import (
	"fmt"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestPageView_AnonymousDistinctSlugsGrowNoRowsBeyondTheOpenableEntries(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "open-one", catalog.VisibilityPublic)
	seed(t, cm, ctx, "pending-one", catalog.VisibilityPending)

	for i := 0; i < 200; i++ {
		do(t, h, "POST", "/page-views", `{"page":"entry","slug":"open-one"}`)
		do(t, h, "POST", "/page-views", `{"page":"entry","slug":"pending-one"}`)
		do(t, h, "POST", "/page-views", fmt.Sprintf(`{"page":"entry","slug":"never-filed-%d"}`, i))
		do(t, h, "POST", "/page-views", fmt.Sprintf(`{"page":"index","slug":"never-filed-%d"}`, i))
	}

	views, err := cm.ListPageViews(ctx)
	if err != nil {
		t.Fatalf("ListPageViews: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("800 anonymous page views over 402 distinct slugs left %d counter rows, want 2 (the index and the one openable entry): %+v", len(views), views)
	}
	for _, v := range views {
		if v.Views != 200 {
			t.Errorf("counter %s/%q reads %d, want 200", v.Page, v.Slug, v.Views)
		}
	}
}
