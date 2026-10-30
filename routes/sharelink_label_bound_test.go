package routes_test

import (
	"net/http"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT32_ShareLinkLabelIsBounded(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "alpha", catalog.VisibilityPrivate)

	long := `{"label":"` + strings.Repeat("l", 64<<10) + `"}`
	if rec := do(t, h, http.MethodPost, "/entries/alpha/links", long); rec.Code != http.StatusBadRequest {
		t.Errorf("a 64 KiB share link label answered %d; want 400", rec.Code)
	}
	if rec := do(t, h, http.MethodPost, "/entries/alpha/links", `{"label":"for a reviewer"}`); rec.Code != http.StatusCreated {
		t.Fatalf("a short label answered %d %s; want 201", rec.Code, rec.Body.String())
	}
}
