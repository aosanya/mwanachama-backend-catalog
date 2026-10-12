package routes_test

import (
	"encoding/json"
	"sort"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAToolGivenNullForEveryRequiredArgumentRefusesAndWritesNothing(t *testing.T) {
	tools, _, ctx := newTools(t)
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)

	gated := 0
	for _, name := range names {
		tool := tools[name]
		required, _ := schema(t, tool)["required"].([]any)
		if len(required) == 0 {
			continue
		}
		gated++
		args := map[string]any{}
		for _, key := range required {
			args[key.(string)] = nil
		}
		raw, err := json.Marshal(args)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		out, err := tool.Invoke(ctx, raw)
		if err == nil {
			t.Errorf("%s with every required argument null answered %+v; it should refuse", name, out)
		}
	}
	if gated < 20 {
		t.Fatalf("precondition: only %d tools declare a required argument", gated)
	}

	entries := invoke(t, tools["catalog_entry_list"], ctx, `{}`).([]catalog.Entry)
	if len(entries) != 0 {
		t.Fatalf("a null-argument call stored entries: %+v", entries)
	}
	suggestions := invoke(t, tools["catalog_suggestion_list"], ctx, `{}`)
	if b, _ := json.Marshal(suggestions); string(b) != "[]" {
		t.Fatalf("a null-argument call stored suggestions: %s", b)
	}
}
