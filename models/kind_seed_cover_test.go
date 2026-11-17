package models

import (
	"errors"
	"testing"
)

func TestParseEntryKindsTreatsNilAsNoKinds(t *testing.T) {
	kinds, err := ParseEntryKinds(nil)
	if err != nil || kinds != nil {
		t.Fatalf("kinds = %v, err = %v", kinds, err)
	}
}

func TestParseEntryKindsReadsTheDeclaredFields(t *testing.T) {
	value := []any{map[string]any{"name": "book", "fields": []any{
		map[string]any{"name": "isbn", "type": "string", "required": true},
		map[string]any{"name": "pages", "type": "number"},
	}}}
	kinds, err := ParseEntryKinds(value)
	if err != nil || len(kinds) != 1 || kinds[0].Name != "book" || len(kinds[0].Fields) != 2 || !kinds[0].Fields[0].Required {
		t.Fatalf("kinds = %+v, err = %v", kinds, err)
	}
}

func TestParseEntryKindsRefusesMalformedSettings(t *testing.T) {
	field := func(name, typ string) map[string]any { return map[string]any{"name": name, "type": typ} }
	kind := func(name string, fields ...any) map[string]any {
		return map[string]any{"name": name, "fields": fields}
	}
	cases := map[string]any{
		"not a list":            "book",
		"unnamed kind":          []any{kind("")},
		"duplicate kind":        []any{kind("a"), kind("a")},
		"unnamed field":         []any{kind("a", field("", "string"))},
		"field called kind":     []any{kind("a", field("kind", "string"))},
		"duplicate field":       []any{kind("a", field("x", "string"), field("x", "number"))},
		"unknown field type":    []any{kind("a", field("x", "date"))},
		"unmarshalable setting": make(chan int),
	}
	for name, value := range cases {
		if _, err := ParseEntryKinds(value); !errors.Is(err, ErrKindsMalformed) {
			t.Fatalf("%s: err = %v", name, err)
		}
	}
}

func TestEntryKindFitsChecksPresenceAndEachValueShape(t *testing.T) {
	kind := EntryKind{Name: "book", Fields: []KindField{
		{Name: "isbn", Type: "string", Required: true},
		{Name: "pages", Type: "number"},
		{Name: "signed", Type: "boolean"},
		{Name: "meta", Type: "object"},
		{Name: "tags", Type: "array"},
	}}
	cases := []struct {
		name string
		doc  map[string]any
		want bool
	}{
		{"only the required field", map[string]any{"isbn": "1"}, true},
		{"every field right", map[string]any{"isbn": "1", "pages": 10.0, "signed": true, "meta": map[string]any{}, "tags": []any{}}, true},
		{"required missing", map[string]any{"pages": 1.0}, false},
		{"required null", map[string]any{"isbn": nil}, false},
		{"optional null is skipped", map[string]any{"isbn": "1", "pages": nil}, true},
		{"string wrong", map[string]any{"isbn": 5.0}, false},
		{"number wrong", map[string]any{"isbn": "1", "pages": "ten"}, false},
		{"boolean wrong", map[string]any{"isbn": "1", "signed": "yes"}, false},
		{"object wrong", map[string]any{"isbn": "1", "meta": []any{}}, false},
		{"array wrong", map[string]any{"isbn": "1", "tags": map[string]any{}}, false},
	}
	for _, c := range cases {
		if got := kind.Fits(c.doc); got != c.want {
			t.Fatalf("%s = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestEntryKindFitsRefusesAFieldWhoseDeclaredTypeIsUnknown(t *testing.T) {
	kind := EntryKind{Fields: []KindField{{Name: "x", Type: "date"}}}
	if kind.Fits(map[string]any{"x": "2026-01-01"}) {
		t.Fatal("unknown type must not hold")
	}
}

func TestParseEntrySeedsTreatsNilAsNoSeedsAndReadsAList(t *testing.T) {
	if seeds, err := ParseEntrySeeds(nil); err != nil || seeds != nil {
		t.Fatalf("nil = %v, %v", seeds, err)
	}
	seeds, err := ParseEntrySeeds([]any{map[string]any{"slug": "a", "name": "A", "summary": "s", "doc": map[string]any{"k": "v"}}})
	if err != nil || len(seeds) != 1 || seeds[0].Summary != "s" {
		t.Fatalf("seeds = %+v, err = %v", seeds, err)
	}
	if entry := seeds[0].Entry(); entry.Doc != `{"k":"v"}` || entry.Slug != "a" || entry.Name != "A" || entry.Summary != "s" {
		t.Fatalf("entry = %+v", entry)
	}
}

func TestParseEntrySeedsRefusesMalformedSettings(t *testing.T) {
	seed := func(slug, name string, doc any) map[string]any {
		m := map[string]any{"slug": slug, "name": name}
		if doc != nil {
			m["doc"] = doc
		}
		return m
	}
	cases := map[string]any{
		"not a list":      "seeds",
		"missing slug":    []any{seed("", "A", nil)},
		"missing name":    []any{seed("a", "", nil)},
		"duplicate slug":  []any{seed("a", "A", nil), seed("a", "B", nil)},
		"doc is an array": []any{seed("a", "A", []any{1})},
		"doc is a string": []any{seed("a", "A", "text")},
		"doc is null":     []any{map[string]any{"slug": "a", "name": "A", "doc": nil}},
		"unknown field":   []any{map[string]any{"slug": "a", "name": "A", "extra": 1}},
		"unmarshalable":   make(chan int),
	}
	for name, value := range cases {
		if _, err := ParseEntrySeeds(value); !errors.Is(err, ErrSeedsMalformed) {
			t.Fatalf("%s: err = %v", name, err)
		}
	}
}

func TestEntrySeedWithoutADocBecomesAnEmptyObject(t *testing.T) {
	if entry := (EntrySeed{Slug: "a", Name: "A"}).Entry(); entry.Doc != "{}" {
		t.Fatalf("doc = %q", entry.Doc)
	}
}
