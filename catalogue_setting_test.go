package mwanachamacatalog_test

import (
	"encoding/json"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCheckSettingsAcceptsDeclaredCatalogues(t *testing.T) {
	raw := json.RawMessage(`{"catalogues":[{"mount":"hr","name":"HR services","audience":"staff","services":true},{"mount":"tours","name":"Tours & Guides","audience":"public"}]}`)
	if problems := catalog.CheckSettings(raw); len(problems) != 0 {
		t.Fatalf("problems = %v, want none", problems)
	}
	if problems := catalog.CheckSettings(nil); len(problems) != 0 {
		t.Fatalf("empty settings: problems = %v, want none", problems)
	}
}

func TestCheckSettingsRefusesABadCatalogueByName(t *testing.T) {
	cases := map[string]string{
		`{"catalogues":[{"mount":"Tours & Guides","name":"Tours","audience":"public"}]}`:                                     "mount",
		`{"catalogues":[{"mount":"hr","name":"HR","audience":"staff"},{"mount":"hr","name":"HR again","audience":"staff"}]}`: "twice",
		`{"catalogues":[{"mount":"hr","name":" ","audience":"staff"}]}`:                                                      "name",
		`{"catalogues":[{"mount":"hr","name":"HR","audience":"everyone"}]}`:                                                  "audience",
		`{"catalogues":[{"mount":"hr","name":"HR","audience":"staff","colour":"red"}]}`:                                      "each catalogue",
		`{"catalogues":{"mount":"hr"}}`: "catalogues",
		`{"shelves":[]}`:                `not declared`,
	}
	for raw, want := range cases {
		problems := catalog.CheckSettings(json.RawMessage(raw))
		if len(problems) == 0 || !strings.Contains(strings.Join(problems, "; "), want) {
			t.Errorf("%s: problems = %v, want one naming %q", raw, problems, want)
		}
	}
}

func TestParseCataloguesReadsServicesAndAudience(t *testing.T) {
	var value any
	if err := json.Unmarshal([]byte(`[{"mount":"it","name":"IT services","audience":"members","services":true}]`), &value); err != nil {
		t.Fatal(err)
	}
	got, problems := catalog.ParseCatalogues(value)
	if len(problems) != 0 || len(got) != 1 || got[0].Audience != catalog.AudienceMembers || !got[0].Services || got[0].Mount != "it" {
		t.Fatalf("got %+v, %v", got, problems)
	}
}

func TestSpecForCarriesNoStorefrontIndexes(t *testing.T) {
	s, err := catalog.SpecFor("agy1", "hr")
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range s.Objects {
		for _, ix := range o.Indexes {
			if ix.Path != nil && strings.HasPrefix(ix.Path.Path, "_template.") {
				t.Errorf("%s carries the storefront's index %q", o.Name, ix.Name)
			}
		}
	}
	if _, declared := s.Setting("catalogues"); !declared {
		t.Error("the catalogues setting is not declared")
	}
}
