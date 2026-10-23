package routes_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-catalog/routes"
)

func newTools(t *testing.T) (map[string]routes.Tool, catalog.CatalogManager, context.Context) {
	t.Helper()
	_, cm, ctx := newServer(t)
	built, err := routes.BuildTools(cm)
	if err != nil {
		t.Fatalf("build tools: %v", err)
	}
	byName := map[string]routes.Tool{}
	for _, tool := range built {
		byName[tool.Name] = tool
	}
	return byName, cm, ctx
}

func invoke(t *testing.T, tool routes.Tool, ctx context.Context, args string) any {
	t.Helper()
	out, err := tool.Invoke(ctx, json.RawMessage(args))
	if err != nil {
		t.Fatalf("%s: %v", tool.Name, err)
	}
	return out
}

func schema(t *testing.T, tool routes.Tool) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(tool.InputSchema, &out); err != nil {
		t.Fatalf("%s schema: %v", tool.Name, err)
	}
	return out
}

func properties(t *testing.T, tool routes.Tool) map[string]any {
	t.Helper()
	props, ok := schema(t, tool)["properties"].(map[string]any)
	if !ok {
		t.Fatalf("%s has no properties", tool.Name)
	}
	return props
}

func TestEveryGatedOperationBecomesATool(t *testing.T) {
	tools, _, _ := newTools(t)
	if len(tools) != 31 {
		t.Fatalf("built %d tools: %v", len(tools), tools)
	}
	for _, want := range []string{
		"catalog_entry_upsert", "catalog_entry_list", "catalog_entry_read",
		"catalog_entry_write_private", "catalog_run_record", "catalog_setting_read",
		"catalog_suggestion_match_attach", "catalog_suggestion_match_list_for_entry", "catalog_raised_item_list_nearby",
	} {
		if _, ok := tools[want]; !ok {
			t.Fatalf("no tool named %s", want)
		}
	}
}

func TestTheReadersOperationsAreNotTools(t *testing.T) {
	tools, _, _ := newTools(t)
	for _, unwanted := range []string{"catalog_entry_open", "catalog_page_view_record"} {
		if _, ok := tools[unwanted]; ok {
			t.Fatalf("%s is an anonymous action and must not be a tool", unwanted)
		}
	}
}

func TestEveryToolCarriesProseAndASchema(t *testing.T) {
	tools, _, _ := newTools(t)
	for name, tool := range tools {
		if tool.Description == "" {
			t.Fatalf("%s has no description", name)
		}
		if tool.Title == "" {
			t.Fatalf("%s has no title", name)
		}
		if schema(t, tool)["type"] != "object" {
			t.Fatalf("%s: schema is not an object", name)
		}
	}
}

func TestTheUpsertToolTakesTheDocumentAndNotTheStoresOwnFields(t *testing.T) {
	tools, _, _ := newTools(t)
	props := properties(t, tools["catalog_entry_upsert"])
	for _, want := range []string{"slug", "name", "summary", "doc"} {
		if _, ok := props[want]; !ok {
			t.Fatalf("property %q is missing: %v", want, props)
		}
	}
	for _, unwanted := range []string{"id", "created_at", "updated_at", "approved_at", "visibility", "deleted", "submitted_by"} {
		if _, ok := props[unwanted]; ok {
			t.Fatalf("the store owns %q, so it must not be an argument: %v", unwanted, props)
		}
	}
	doc, _ := props["doc"].(map[string]any)
	if !strings.Contains(doc["description"].(string), "whole document") {
		t.Fatalf("doc carries no blueprint prose: %v", doc)
	}
}

func TestVisibilityIsSetByItsOwnToolWithTheDeclaredValues(t *testing.T) {
	tools, _, _ := newTools(t)
	visibility, _ := properties(t, tools["catalog_entry_set_visibility"])["visibility"].(map[string]any)
	values, _ := visibility["enum"].([]any)
	if len(values) != 3 {
		t.Fatalf("visibility = %v", visibility)
	}
	if values[0] != "pending" || values[1] != "public" || values[2] != "private" {
		t.Fatalf("visibility = %v", visibility)
	}
}

func TestAMinedEntryReachesTheStoreThroughItsTool(t *testing.T) {
	tools, cm, ctx := newTools(t)
	invoke(t, tools["catalog_entry_upsert"], ctx,
		`{"slug":"county-health","name":"County health authority","summary":"A health authority.","doc":`+
			strconv.Quote(doc)+`}`)

	stored, err := cm.GetEntry(ctx, "county-health")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.Name != "County health authority" {
		t.Fatalf("stored = %+v", stored)
	}
	if stored.Visibility != models.VisibilityPending {
		t.Fatalf("a tool published an entry: %v", stored.Visibility)
	}
	if stored.Doc != doc {
		t.Fatalf("doc = %q", stored.Doc)
	}
}

func TestAListingToolFiltersOnADocumentPath(t *testing.T) {
	tools, cm, ctx := newTools(t)
	if _, err := cm.UpsertEntry(ctx, catalog.Entry{Slug: "one", Name: "One", Doc: doc}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	out := invoke(t, tools["catalog_entry_list"], ctx,
		`{"docs":[{"path":"_template.sector","value":"health"}],"limit":10}`)
	cards, ok := out.([]catalog.Entry)
	if !ok {
		t.Fatalf("out = %T", out)
	}
	if len(cards) != 1 || cards[0].Slug != "one" {
		t.Fatalf("cards = %+v", cards)
	}

	none := invoke(t, tools["catalog_entry_list"], ctx,
		`{"docs":[{"path":"_template.sector","value":"transport"}]}`)
	if len(none.([]catalog.Entry)) != 0 {
		t.Fatalf("none = %+v", none)
	}
}

func TestAToolRefusesAnArgumentTheOperationDoesNotDeclare(t *testing.T) {
	tools, _, ctx := newTools(t)
	_, err := tools["catalog_entry_upsert"].Invoke(ctx, json.RawMessage(`{"slug":"one","name":"One","sector":"health"}`))
	if err == nil || !strings.Contains(err.Error(), `takes no argument "sector"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestAnUnknownEntryIsTheSameRefusalItIsOverHTTP(t *testing.T) {
	tools, _, ctx := newTools(t)
	_, err := tools["catalog_entry_read"].Invoke(ctx, json.RawMessage(`{"slug":"nothing-here"}`))
	if err == nil || !strings.Contains(err.Error(), catalog.ErrEntryNotFound.Error()) {
		t.Fatalf("err = %v", err)
	}
}

func TestEveryDescribedFieldIsOneItsTypeCarries(t *testing.T) {
	fields, err := routes.Fields()
	if err != nil {
		t.Fatalf("fields: %v", err)
	}
	carriers := map[string]reflect.Type{
		"Entry":             reflect.TypeOf(models.Entry{}),
		"EntryPrivate":      reflect.TypeOf(models.EntryPrivate{}),
		"ShareLink":         reflect.TypeOf(models.ShareLink{}),
		"Suggestion":        reflect.TypeOf(models.Suggestion{}),
		"SuggestionComment": reflect.TypeOf(models.SuggestionComment{}),
		"SuggestionMatch":   reflect.TypeOf(models.SuggestionMatch{}),
		"MiningRun":         reflect.TypeOf(models.MiningRun{}),
		"Setting":           reflect.TypeOf(models.Setting{}),
		"PageView":          reflect.TypeOf(models.PageView{}),
		"Record":            reflect.TypeOf(models.Record{}),
		"Endorsement":       reflect.TypeOf(models.Endorsement{}),
		"RaisedItem":        reflect.TypeOf(models.RaisedItem{}),
		"BoardItem":         reflect.TypeOf(models.BoardItem{}),
		"Reply":             reflect.TypeOf(models.Reply{}),
		"Removal":           reflect.TypeOf(models.Removal{}),
		"RaisedItemFile":    reflect.TypeOf(models.RaisedItemFile{}),
		"ListFilter":        reflect.TypeOf(catalog.ListFilter{}),
	}
	for key := range fields {
		carrier, field, _ := strings.Cut(key, ".")
		t2, ok := carriers[carrier]
		if !ok {
			t.Fatalf("%q describes no type this module carries", key)
		}
		if !holds(t2, field) {
			t.Fatalf("%s has no field reaching the wire as %q", carrier, field)
		}
	}
}

func holds(t reflect.Type, wire string) bool {
	want := plain(wire)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if plain(f.Name) == want {
			return true
		}
		tag, tagged := f.Tag.Lookup("json")
		if tagged && plain(strings.Split(tag, ",")[0]) == want {
			return true
		}
	}
	return false
}

func plain(s string) string { return strings.ToLower(strings.ReplaceAll(s, "_", "")) }
