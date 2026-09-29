package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const statusFloodSize = 40000

var driverWords = []string{"SQL", "sql", "variables", "sqlite", "pq:", "SELECT", "select "}

func namesTheDriver(s string) string {
	for _, w := range driverWords {
		if strings.Contains(s, w) {
			return w
		}
	}
	return ""
}

func TestAFloodOfRepeatedStatusesThatBreaksTheStoreNamesNothingAboutIt(t *testing.T) {
	h, _, _ := newServer(t)

	one := do(t, h, "GET", "/suggestions?status=proposed", "")
	if one.Code != http.StatusOK {
		t.Fatalf("precondition: one status answered %d %s", one.Code, one.Body.String())
	}

	query := strings.Repeat("status=proposed&", statusFloodSize)
	if len(query) >= http.DefaultMaxHeaderBytes {
		t.Fatalf("precondition: the flood is %d bytes, past what a real server would even read", len(query))
	}
	flood := do(t, h, "GET", "/suggestions?"+query, "")
	if flood.Code == http.StatusOK {
		t.Fatalf("precondition: GET /suggestions with %d statuses answered 200, so the store's error path was never reached", statusFloodSize)
	}
	if w := namesTheDriver(flood.Body.String()); w != "" {
		t.Errorf("GET /suggestions with %d statuses answered %d naming %q: %s; want a refusal that says nothing about the database", statusFloodSize, flood.Code, w, flood.Body.String())
	}

	tools, _, _ := newTools(t)
	list, ok := tools["catalog_suggestion_list"]
	if !ok {
		t.Fatal("precondition: no tool named catalog_suggestion_list is built")
	}
	statuses := "[" + strings.TrimSuffix(strings.Repeat(`"proposed",`, statusFloodSize), ",") + "]"
	_, err := list.Invoke(context.Background(), json.RawMessage(`{"status":`+statuses+`}`))
	if err == nil {
		t.Fatalf("precondition: catalog_suggestion_list with %d statuses succeeded, so the store's error path was never reached", statusFloodSize)
	}
	if w := namesTheDriver(err.Error()); w != "" {
		t.Errorf("catalog_suggestion_list with %d statuses failed naming %q: %s; want a refusal that says nothing about the database", statusFloodSize, w, err.Error())
	}
}
