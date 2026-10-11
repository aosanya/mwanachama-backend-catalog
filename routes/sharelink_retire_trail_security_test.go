package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func retiredAtOf(t *testing.T, h http.Handler, slug, linkID string) string {
	t.Helper()
	rec := do(t, h, http.MethodGet, "/entries/"+slug+"/links", "")
	var links []catalog.ShareLink
	if err := json.Unmarshal(rec.Body.Bytes(), &links); err != nil {
		t.Fatalf("listing %s's links answered %d %s", slug, rec.Code, rec.Body.String())
	}
	for _, l := range links {
		if l.ID == linkID {
			return l.RetiredAt
		}
	}
	t.Fatalf("link %s is missing from %s's listing %s", linkID, slug, rec.Body.String())
	return ""
}

func TestRetiringAShareLinkAgainNeverRewritesWhenItStoppedWorking(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "trail", catalog.VisibilityPrivate)
	seed(t, cm, ctx, "bulk", catalog.VisibilityPrivate)

	key, linkID := issueKey(t, h, "trail")
	if rec := do(t, h, http.MethodPost, "/links/"+linkID+"/retire", ""); rec.Code != http.StatusOK {
		t.Fatalf("precondition: first retire answered %d %s", rec.Code, rec.Body.String())
	}
	first := retiredAtOf(t, h, "trail", linkID)
	if first == "" {
		t.Fatalf("precondition: a retired link carries no retired_at")
	}

	_, bulkID := issueKey(t, h, "bulk")
	if _, err := cm.SetVisibility(ctx, "bulk", catalog.VisibilityPublic); err != nil {
		t.Fatalf("precondition: making bulk public: %v", err)
	}
	bulkFirst := retiredAtOf(t, h, "bulk", bulkID)
	if bulkFirst == "" {
		t.Fatalf("precondition: leaving private left bulk's link with no retired_at")
	}

	time.Sleep(1100 * time.Millisecond)

	again := do(t, h, http.MethodPost, "/links/"+linkID+"/retire", "")
	if again.Code != http.StatusOK {
		t.Errorf("retiring an already-retired link answered %d %s; want 200 with the stored link", again.Code, again.Body.String())
	}
	if got := retiredAtOf(t, h, "trail", linkID); got != first {
		t.Errorf("a second retire moved retired_at from %s to %s; the trail a rotation leaves must record when the key first stopped working", first, got)
	}
	if rec := do(t, h, http.MethodPost, "/links/"+bulkID+"/retire", ""); rec.Code != http.StatusOK {
		t.Errorf("retiring a link already retired by leaving private answered %d %s; want 200", rec.Code, rec.Body.String())
	}
	if got := retiredAtOf(t, h, "bulk", bulkID); got != bulkFirst {
		t.Errorf("an explicit retire after leaving private moved retired_at from %s to %s; want it unchanged", bulkFirst, got)
	}
	if rec := do(t, h, http.MethodGet, "/entries/trail?k="+key, ""); rec.Code != http.StatusNotFound {
		t.Errorf("a twice-retired key answered %d %s; want 404", rec.Code, rec.Body.String())
	}
}
