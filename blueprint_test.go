package mwanachamacatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-catalog/spec"
)

func TestVocabularyMatchesTheBlueprint(t *testing.T) {
	inGo := map[string][]string{
		roleEntry + ".visibility": {
			string(models.VisibilityPending),
			string(models.VisibilityPublic),
			string(models.VisibilityPrivate),
		},
		roleShareLink + ".status": {
			string(models.LinkStatusActive),
			string(models.LinkStatusRetired),
		},
		roleSuggestion + ".source": {
			string(models.SuggestionSourceRun),
			string(models.SuggestionSourceOwner),
			string(models.SuggestionSourceStray),
		},
		roleSuggestion + ".status": {
			string(models.SuggestionStatusProposed),
			string(models.SuggestionStatusApproved),
			string(models.SuggestionStatusRejected),
			string(models.SuggestionStatusBuilt),
			string(models.SuggestionStatusStalled),
		},
		roleSuggestionComment + ".author": {
			string(models.CommentAuthorRun),
			string(models.CommentAuthorAdmin),
		},
		rolePageView + ".page": {
			string(models.PageKindIndex),
			string(models.PageKindEntry),
		},
	}

	b, err := Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}

	for _, o := range b.Objects {
		for _, f := range o.Fields {
			if f.Type != spec.TypeEnum {
				continue
			}
			key := o.Role + "." + f.Name
			want, named := inGo[key]
			if !named {
				t.Errorf("the blueprint declares the enum %s with the values %s, and Go names no constants for them",
					key, strings.Join(f.Values, ", "))
				continue
			}
			delete(inGo, key)
			if !sameValues(want, f.Values) {
				t.Errorf("%s: Go names %s, the blueprint declares %s — a stored value outlives a rename, so these have to agree",
					key, strings.Join(sorted(want), ", "), strings.Join(sorted(f.Values), ", "))
			}
		}
	}

	for key := range inGo {
		t.Errorf("Go names constants for %s, which the blueprint declares no enum for", key)
	}
}

func TestEveryExampleFitsTheTypes(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("spec", "examples", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("glob examples: %v (%d found)", err, len(paths))
	}

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			s, err := LoadSpec(path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			db := openTestDB(t)
			if err := spec.Migrate(db, s); err != nil {
				t.Fatalf("migrate: %v", err)
			}
			if _, err := NewCatalogManager(db, s); err != nil {
				t.Fatalf("manager: %v", err)
			}
		})
	}
}

func TestExamplesDeclareNoFieldsOfTheirOwn(t *testing.T) {
	paths, _ := filepath.Glob(filepath.Join("spec", "examples", "*.json"))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var doc struct {
			Objects []struct {
				Name   string `json:"name"`
				Role   string `json:"role"`
				Fields []struct {
					Name        string   `json:"name"`
					Type        string   `json:"type"`
					Description string   `json:"description"`
					Values      []string `json:"values"`
				} `json:"fields"`
			} `json:"objects"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, o := range doc.Objects {
			if o.Role == "" {
				continue
			}
			for _, f := range o.Fields {
				if f.Type != "" || f.Description != "" || len(f.Values) > 0 {
					t.Errorf("%s: %s restates the declaration of %q — a domain names the object and sets defaults, the module declares the field",
						filepath.Base(path), o.Name, f.Name)
				}
			}
		}
	}
}

func sameValues(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := sorted(a), sorted(b)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

func sorted(in []string) []string {
	out := append([]string{}, in...)
	sort.Strings(out)
	return out
}
