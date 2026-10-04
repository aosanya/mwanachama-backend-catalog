package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAResendCannotSmuggleAChangedDocumentPastItsReview(t *testing.T) {
	put := func(t *testing.T, h http.Handler, d string) int {
		body, _ := json.Marshal(map[string]any{"name": "Agency", "doc": d})
		return do(t, h, "PUT", "/entries/pub", string(body)).Code
	}

	t.Run("reordered and respaced stays published", func(t *testing.T) {
		h, cm, _ := newMountedServer(t)
		seed(t, cm, context.Background(), "pub", catalog.VisibilityPublic)
		if code := put(t, h, `{ "modules":[{"enabled":true,"code":"auth"}], "_template":{"sector":"health"}, "agency":{"name":"Test"} }`); code != http.StatusOK {
			t.Fatalf("precondition: re-sending the same document reordered answered %d", code)
		}
		if got := do(t, h, "GET", "/entries/pub", "").Code; got != http.StatusOK {
			t.Fatalf("precondition: a semantically unchanged re-send took the entry off the public read (%d)", got)
		}
	})

	t.Run("trailing data is refused and the reviewed document stands", func(t *testing.T) {
		h, cm, _ := newMountedServer(t)
		seed(t, cm, context.Background(), "pub", catalog.VisibilityPublic)
		smuggled := doc + ` {"agency":{"name":"Unreviewed"}}`
		if code := put(t, h, smuggled); code != http.StatusBadRequest {
			t.Errorf("a re-send whose document carries a second JSON value after the reviewed one answered %d; want 400", code)
		}
		e, err := cm.GetEntry(context.Background(), "pub")
		if err != nil {
			t.Fatalf("GetEntry: %v", err)
		}
		if e.Doc != doc || e.Visibility != catalog.VisibilityPublic {
			t.Errorf("after a refused trailing-data re-send the entry reads visibility=%s doc=%q; want public and the reviewed document unchanged", e.Visibility, e.Doc)
		}
	})

	for name, changed := range map[string]string{
		"a key differing only in case": `{"Agency":{"name":"Test"},"_template":{"sector":"health"},"modules":[{"code":"auth","enabled":true}]}`,
		"a number written differently": `{"agency":{"name":"Test"},"_template":{"sector":"health"},"modules":[{"code":"auth","enabled":true}],"n":1.0}`,
	} {
		t.Run(name+" returns to pending", func(t *testing.T) {
			h, cm, _ := newMountedServer(t)
			seed(t, cm, context.Background(), "pub", catalog.VisibilityPublic)
			if code := put(t, h, changed); code != http.StatusOK {
				t.Fatalf("precondition: re-sending a changed document answered %d", code)
			}
			if got := do(t, h, "GET", "/entries/pub", "").Code; got != http.StatusNotFound {
				t.Errorf("a re-send with %s left the entry on the anonymous read (%d); want 404 until a reviewer approves it again", name, got)
			}
		})
	}
}
