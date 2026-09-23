package spec_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-catalog/spec"
)

const oneObject = `{
  "module": "catalog",
  "objects": [
    {
      "role": "entry",
      "description": "One catalog document.",
      "fields": [
        {"name": "id", "type": "string", "description": "Storage key.", "primary": true},
        {"name": "slug", "type": "string", "description": "URL key.", "required": true},
        {"name": "visibility", "type": "enum", "description": "Who may read it.",
         "default": "pending", "values": ["pending", "public", "private"]}
      ],
      "indexes": [
        {"name": "visibility", "description": "The listing's filter.", "fields": ["visibility"]}
      ]
    }
  ]
}`

func domainSpec(objects string) string {
	return `{"module":"catalog","domain":"agency","instance":"agency","objects":[` + objects + `]}`
}

func mustBlueprint(t *testing.T, raw string) *spec.Blueprint {
	t.Helper()
	b, err := spec.ParseBlueprint([]byte(raw))
	if err != nil {
		t.Fatalf("parse blueprint: %v", err)
	}
	return b
}

func TestBlueprintFillsTheFieldsADomainDoesNotRestate(t *testing.T) {
	b := mustBlueprint(t, oneObject)
	s, err := b.Parse([]byte(domainSpec(
		`{"role":"entry","name":"agency","table":"agencies","description":"An archetype."}`)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	o, ok := s.ByRole("entry")
	if !ok {
		t.Fatal("no object fills the entry role")
	}
	if len(o.Fields) != 3 {
		t.Fatalf("fields = %d, want the 3 the module declares", len(o.Fields))
	}
	if o.Name != "agency" || o.Table != "agencies" {
		t.Errorf("name/table = %q/%q, want the domain's own", o.Name, o.Table)
	}
	if s.TableFor(o) != "agency_catalog_agencies" {
		t.Errorf("TableFor = %q", s.TableFor(o))
	}
	if len(o.Indexes) != 1 || o.Indexes[0].Name != "visibility" {
		t.Errorf("indexes = %v, want the module's own carried over", o.Indexes)
	}
}

func TestBlueprintDomainSetsADefaultAndNothingElse(t *testing.T) {
	b := mustBlueprint(t, oneObject)

	t.Run("a default is taken", func(t *testing.T) {
		s, err := b.Parse([]byte(domainSpec(
			`{"role":"entry","name":"agency","table":"agencies","description":"An archetype.",
			  "fields":[{"name":"visibility","default":"public"}]}`)))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		o, _ := s.ByRole("entry")
		for _, f := range o.Fields {
			if f.Name == "visibility" {
				if f.Default != "public" {
					t.Errorf("default = %q, want the domain's", f.Default)
				}
				if len(f.Values) != 3 {
					t.Errorf("values = %v, want the module's kept", f.Values)
				}
			}
		}
	})

	for _, tc := range []struct{ name, field, want string }{
		{"a type", `{"name":"visibility","type":"string"}`, "may set a default and nothing else"},
		{"a description", `{"name":"slug","description":"mine now"}`, "may set a default and nothing else"},
		{"more values", `{"name":"visibility","values":["pending","embargoed"]}`, "may set a default and nothing else"},
		{"a field the module does not declare", `{"name":"sector","type":"string","description":"x"}`, "declares no field"},
	} {
		t.Run("refuses "+tc.name, func(t *testing.T) {
			_, err := b.Parse([]byte(domainSpec(
				`{"role":"entry","name":"agency","table":"agencies","description":"An archetype.",
				  "fields":[` + tc.field + `]}`)))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestBlueprintRefusesARoleTheModuleDoesNotDeclare(t *testing.T) {
	b := mustBlueprint(t, oneObject)
	_, err := b.Parse([]byte(domainSpec(
		`{"role":"loan","name":"loan","table":"loans","description":"A loan."}`)))
	if err == nil || !strings.Contains(err.Error(), "which the module does not declare") {
		t.Fatalf("err = %v, want a refusal naming the unknown role", err)
	}
}

func TestBlueprintLeavesADomainsOwnObjectAlone(t *testing.T) {
	b := mustBlueprint(t, oneObject)
	s, err := b.Parse([]byte(domainSpec(
		`{"role":"entry","name":"agency","table":"agencies","description":"An archetype."},
		 {"name":"copy","table":"copies","description":"One physical copy.",
		  "fields":[{"name":"id","type":"string","description":"Storage key.","primary":true},
		            {"name":"shelfmark","type":"string","description":"Where it sits.","required":true}]}`)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	o, ok := s.Object("copy")
	if !ok {
		t.Fatal("the domain's own object is gone")
	}
	if len(o.Fields) != 2 {
		t.Errorf("fields = %d, want the domain's own 2", len(o.Fields))
	}
}

func TestBlueprintRefusesWhatIsTheDomainsToChoose(t *testing.T) {
	for _, tc := range []struct{ name, object, want string }{
		{"a name", `{"role":"entry","name":"agency","description":"x","fields":[{"name":"id","type":"string","description":"k","primary":true}]}`,
			`declares the name "agency", which is the domain's to choose`},
		{"a table", `{"role":"entry","table":"agencies","description":"x","fields":[{"name":"id","type":"string","description":"k","primary":true}]}`,
			`declares the table "agencies", which is the domain's to choose`},
		{"no role at all", `{"description":"x","fields":[{"name":"id","type":"string","description":"k","primary":true}]}`,
			"declares no role"},
	} {
		t.Run("refuses "+tc.name, func(t *testing.T) {
			_, err := spec.ParseBlueprint([]byte(`{"module":"catalog","objects":[` + tc.object + `]}`))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestBlueprintRefusesASpecForAnotherModule(t *testing.T) {
	b := mustBlueprint(t, oneObject)
	_, err := b.Parse([]byte(`{"module":"forms","domain":"agency","instance":"agency","objects":[
	  {"role":"entry","name":"agency","table":"agencies","description":"An archetype."}]}`))
	if err == nil || !strings.Contains(err.Error(), "this is the blueprint for") {
		t.Fatalf("err = %v, want a refusal naming the module", err)
	}
}

func TestADefaultMustBeOneOfTheDeclaredValues(t *testing.T) {
	b := mustBlueprint(t, oneObject)
	_, err := b.Parse([]byte(domainSpec(
		`{"role":"entry","name":"agency","table":"agencies","description":"An archetype.",
		  "fields":[{"name":"visibility","default":"embargoed"}]}`)))
	if err == nil || !strings.Contains(err.Error(), "which is not one of") {
		t.Fatalf("err = %v, want the default measured against the declared values", err)
	}
}

func TestARequiredFieldMayNotBeGivenADefault(t *testing.T) {
	b := mustBlueprint(t, oneObject)
	_, err := b.Parse([]byte(domainSpec(
		`{"role":"entry","name":"agency","table":"agencies","description":"An archetype.",
		  "fields":[{"name":"slug","default":"untitled"}]}`)))
	if err == nil || !strings.Contains(err.Error(), "pass unnoticed") {
		t.Fatalf("err = %v, want a required field's default refused", err)
	}
}

func TestASpecThatNeedsTheBlueprintSaysSoWhenLoadedWithout(t *testing.T) {
	_, err := spec.Parse([]byte(domainSpec(
		`{"role":"entry","name":"agency","table":"agencies","description":"An archetype."}`)))
	if err == nil || !strings.Contains(err.Error(), "through the module's blueprint") {
		t.Fatalf("err = %v, want the error to name the blueprint", err)
	}
}

func TestShippedExamplesLoadThroughTheShippedBlueprint(t *testing.T) {
	b := blueprint(t)
	for _, path := range []string{"examples/agency.catalog.json", "examples/library.catalog.json"} {
		if _, err := b.Load(path); err != nil {
			t.Errorf("load %s: %v", path, err)
		}
	}
	if _, err := spec.LoadBlueprint(filepath.Join("..", "catalog.blueprint.json")); err != nil {
		t.Errorf("the shipped blueprint does not load: %v", err)
	}
}
