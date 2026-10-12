package mwanachamacatalog_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func libraryWithAttach(t *testing.T, attach ...map[string]string) []byte {
	t.Helper()
	raw, err := os.ReadFile("spec/examples/library.catalog.json")
	if err != nil {
		t.Fatalf("read library spec: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode library spec: %v", err)
	}
	m["attach"] = attach
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("encode library spec: %v", err)
	}
	return out
}

func TestADomainAttachmentReachesOnlyARoleThisCatalogSpecClaims(t *testing.T) {
	s, err := catalog.ParseSpec(libraryWithAttach(t, map[string]string{"module": "ratings", "items": "catalog.entry"}))
	if err != nil {
		t.Fatalf("precondition: a well-formed attachment to catalog.entry must load, got %v", err)
	}
	if got := s.Attached("ratings"); len(got) != 1 || got[0].ItemsRole() != "entry" {
		t.Fatalf("precondition: the loaded spec must carry the one attachment, got %+v", s.Attach)
	}

	refused := map[string]struct {
		attach map[string]string
		says   string
	}{
		"another module's objects": {map[string]string{"module": "ratings", "items": "permissions.grant"}, "points at this spec's own objects"},
		"a roleless domain object": {map[string]string{"module": "ratings", "items": "catalog.copy"}, "no object in this spec claims"},
		"catalog onto itself":      {map[string]string{"module": "catalog", "items": "catalog.entry"}, "cannot attach to itself"},
		"a module off alphabet":    {map[string]string{"module": "ratings;drop", "items": "catalog.entry"}, "not a usable name segment"},
		"a path in items":          {map[string]string{"module": "ratings", "items": "catalog.entry/../share_link"}, "not <module>.<role>"},
	}
	for name, c := range refused {
		_, err := catalog.ParseSpec(libraryWithAttach(t, c.attach))
		if err == nil || !strings.Contains(err.Error(), c.says) {
			t.Errorf("%s: catalog.ParseSpec answered %v; a domain attachment must be refused by name (%q) rather than handed to a host to mount", name, err, c.says)
		}
	}
}
