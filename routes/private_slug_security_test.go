package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestPutPrivateNeverWritesTheEntryTheBodyNames(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "alpha", catalog.VisibilityPrivate)
	seed(t, cm, ctx, "beta", catalog.VisibilityPrivate)

	if rec := do(t, h, "PUT", "/entries/beta/private", `{"readme":"beta-secret","rank":1}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: writing beta's private parts answered %d %s", rec.Code, rec.Body.String())
	}

	const forgedClock = "1999-01-01T00:00:00Z"
	if rec := do(t, h, "PUT", "/entries/alpha/private", `{"slug":"beta","readme":"written-through-alpha","rank":2,"updated_at":"`+forgedClock+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: writing alpha's private parts answered %d %s", rec.Code, rec.Body.String())
	}

	read := func(slug string) catalog.EntryPrivate {
		t.Helper()
		rec := do(t, h, "GET", "/entries/"+slug+"/private", "")
		var p catalog.EntryPrivate
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("reading %s's private parts answered %d %s (%v)", slug, rec.Code, rec.Body.String(), err)
		}
		return p
	}

	beta := read("beta")
	if beta.Readme != "beta-secret" || beta.Rank != 1 {
		t.Errorf("a write to /entries/alpha/private whose body named slug beta changed beta's private parts to %+v; the path must decide which entry is written", beta)
	}
	alpha := read("alpha")
	if alpha.Slug != "alpha" || alpha.Readme != "written-through-alpha" {
		t.Errorf("alpha's private parts read %+v, want the write addressed by the path to land on alpha", alpha)
	}
	if alpha.UpdatedAt == forgedClock {
		t.Errorf("alpha's private parts carry updated_at %q, the value the body claimed; the clock must be the server's", alpha.UpdatedAt)
	}

	missing := do(t, h, "PUT", "/entries/nope/private", `{"slug":"beta","readme":"x"}`)
	if missing.Code != http.StatusNotFound {
		t.Errorf("writing private parts for a missing slug whose body names beta answered %d %s, want 404", missing.Code, missing.Body.String())
	}
	if again := read("beta"); again.Readme != "beta-secret" {
		t.Errorf("a refused write for a missing slug still changed beta's private parts to %+v", again)
	}
}
