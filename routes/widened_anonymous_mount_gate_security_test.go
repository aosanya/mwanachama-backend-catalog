package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-catalog/routes"
)

func TestAMountThatWidensTheAnonymousListStillGatesEveryOtherRouteByItsOwnAction(t *testing.T) {
	f := newDenyAllFixture(t)
	gate := &denyAll{}
	m := routes.Mount{
		Authorize: gate.authorize,
		Caller:    func(context.Context) string { return mountedCaller },
		Anonymous: municipalityAnonymous,
	}
	gated := routes.OperatorRoutesFor(f.cm, m)
	public := routes.PublicRoutesFor(f.cm, m)
	if len(public) != len(municipalityAnonymous) {
		t.Fatalf("precondition: the widened mount opened %d anonymous routes; want %d", len(public), len(municipalityAnonymous))
	}
	if len(gated) == 0 {
		t.Fatal("precondition: the widened mount built no gated routes")
	}
	anonymous := map[string]bool{}
	for _, a := range municipalityAnonymous {
		anonymous[a] = true
	}
	fill := strings.NewReplacer("{slug}", "seeded", "{key}", "k", "{suggestionID}", f.suggestionID, "{linkID}", f.linkID)
	body := `{"visibility":"public","status":"approved","name":"n","doc":` + jsonString(doc) + `,"slug":"wanted2","title":"t","run_id":"r","comment":"c","body":"b","value":"v"}`
	for _, rt := range gated {
		door := rt.Method + " " + fill.Replace(rt.Path)
		if anonymous[rt.Action] {
			t.Errorf("%s carries anonymous action %q but was also built as a gated route", door, rt.Action)
		}
		mux := http.NewServeMux()
		mux.Handle(rt.Method+" "+rt.Path, rt.Handler)
		gate.asked = nil
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(rt.Method, fill.Replace(rt.Path), strings.NewReader(body)))
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s on a mount opening %v to anonymous callers answered %d %s under a deny-all authorizer; want 403", door, municipalityAnonymous, rec.Code, rec.Body.String())
		}
		if len(gate.asked) != 1 || gate.asked[0] != rt.Action {
			t.Errorf("%s asked the authorizer for %v; want exactly its own declared action %q", door, gate.asked, rt.Action)
		}
		f.assertUntouched(t, door)
	}
	for _, rt := range public {
		if !anonymous[rt.Action] {
			t.Errorf("%s %s opened to anonymous callers with action %q, which the mount never named", rt.Method, rt.Path, rt.Action)
		}
	}
}
