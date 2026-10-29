package routes_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-catalog/routes"
	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type denyAll struct{ asked []string }

func (d *denyAll) authorize(_ context.Context, action string) error {
	d.asked = append(d.asked, action)
	return dispatch.ErrForbidden
}

type denyAllFixture struct {
	cm           catalog.CatalogManager
	suggestionID string
	linkID       string
}

func newDenyAllFixture(t *testing.T) denyAllFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s, err := catalog.LoadSpec(filepath.Join("..", "spec", "examples", "agency.catalog.json"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	ctx := context.Background()
	for _, slug := range []string{"seeded", "hidden"} {
		if _, err := cm.UpsertEntry(ctx, models.Entry{Slug: slug, Name: slug, Doc: doc, SubmittedBy: "setup"}); err != nil {
			t.Fatalf("precondition: seeding entry %q failed: %v", slug, err)
		}
	}
	if _, err := cm.SetVisibility(ctx, "hidden", models.VisibilityPrivate); err != nil {
		t.Fatalf("precondition: making the second entry private failed: %v", err)
	}
	link, _, err := cm.IssueShareLink(ctx, "hidden", "pilot", "setup")
	if err != nil {
		t.Fatalf("precondition: issuing a share link failed: %v", err)
	}
	sug, err := cm.FileSuggestion(ctx, models.Suggestion{Slug: "wanted", Title: "Wanted", Source: models.SuggestionSourceOwner, Proposal: `{}`})
	if err != nil {
		t.Fatalf("precondition: filing a suggestion failed: %v", err)
	}
	return denyAllFixture{cm: cm, suggestionID: sug.ID, linkID: link.ID}
}

func (f denyAllFixture) assertUntouched(t *testing.T, door string) {
	t.Helper()
	ctx := context.Background()
	e, err := f.cm.GetEntry(ctx, "seeded")
	if err != nil || e.Visibility != models.VisibilityPending || e.SubmittedBy != "setup" {
		t.Fatalf("%s: a refused call changed the seeded entry: visibility=%q submitted_by=%q err=%v", door, e.Visibility, e.SubmittedBy, err)
	}
	sug, err := f.cm.GetSuggestion(ctx, f.suggestionID)
	if err != nil || sug.Status != models.SuggestionStatusProposed || sug.Attempts != 0 {
		t.Fatalf("%s: a refused call changed the seeded suggestion: status=%q attempts=%d err=%v", door, sug.Status, sug.Attempts, err)
	}
	links, err := f.cm.ListShareLinks(ctx, "hidden")
	if err != nil || len(links) != 1 || links[0].RetiredAt != "" {
		t.Fatalf("%s: a refused call changed the share links: %+v err=%v", door, links, err)
	}
	if runs, _ := f.cm.ListRuns(ctx, 0); len(runs) != 0 {
		t.Fatalf("%s: a refused call recorded %d run(s)", door, len(runs))
	}
	if _, err := f.cm.GetSetting(ctx, "k"); !errors.Is(err, catalog.ErrSettingNotFound) {
		t.Fatalf("%s: a refused call wrote a setting (err=%v)", door, err)
	}
	if comments, _ := f.cm.ListSuggestionComments(ctx, f.suggestionID); len(comments) != 0 {
		t.Fatalf("%s: a refused call wrote %d comment(s)", door, len(comments))
	}
}

func TestEveryGatedToolAsksForItsOwnActionAndWritesNothingWhenRefused(t *testing.T) {
	f := newDenyAllFixture(t)
	gate := &denyAll{}
	tools := routes.ToolsFor(f.cm, routes.Mount{Authorize: gate.authorize, Caller: func(context.Context) string { return mountedCaller }})
	if len(tools) == 0 {
		t.Fatal("precondition: the mount built no tools")
	}
	args := `{"slug":"seeded","key":"k","value":"v","id":"` + f.suggestionID + `","link_id":"` + f.linkID +
		`","suggestion_id":"` + f.suggestionID + `","visibility":"public","status":"approved","name":"n","doc":` +
		jsonString(doc) + `,"title":"t","run_ref":"r","comment":"c","body":"b","author":"operator"}`
	for _, tool := range tools {
		gate.asked = nil
		out, err := tool.Invoke(context.Background(), json.RawMessage(args))
		if !errors.Is(err, dispatch.ErrForbidden) {
			t.Errorf("%s under a deny-all authorizer answered out=%v err=%v; want dispatch.ErrForbidden", tool.Name, out, err)
		}
		if len(gate.asked) != 1 || gate.asked[0] != tool.Action {
			t.Errorf("%s asked the authorizer for %v; want exactly its own declared action %q", tool.Name, gate.asked, tool.Action)
		}
		f.assertUntouched(t, tool.Name)
	}
}

func TestEveryGatedRouteAsksForItsOwnActionAndWritesNothingWhenRefused(t *testing.T) {
	f := newDenyAllFixture(t)
	gate := &denyAll{}
	gated := routes.OperatorRoutesFor(f.cm, routes.Mount{Authorize: gate.authorize, Caller: func(context.Context) string { return mountedCaller }})
	if len(gated) == 0 {
		t.Fatal("precondition: the mount built no gated routes")
	}
	fill := strings.NewReplacer("{slug}", "seeded", "{key}", "k", "{suggestionID}", f.suggestionID, "{linkID}", f.linkID)
	body := `{"visibility":"public","status":"approved","name":"n","doc":` + jsonString(doc) + `,"slug":"wanted2","title":"t","run_ref":"r","comment":"c","body":"b","value":"v"}`
	for _, rt := range gated {
		mux := http.NewServeMux()
		mux.Handle(rt.Method+" "+rt.Path, rt.Handler)
		path := fill.Replace(rt.Path)
		gate.asked = nil
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(rt.Method, path, strings.NewReader(body)))
		door := rt.Method + " " + path
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s under a deny-all authorizer answered %d %s; want 403", door, rec.Code, rec.Body.String())
		}
		if len(gate.asked) != 1 || gate.asked[0] != rt.Action {
			t.Errorf("%s asked the authorizer for %v; want exactly its own declared action %q", door, gate.asked, rt.Action)
		}
		f.assertUntouched(t, door)
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
