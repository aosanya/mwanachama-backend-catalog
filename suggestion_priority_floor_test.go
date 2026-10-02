package mwanachamacatalog_test

import (
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT38_OpenBug_DecideSuggestionCannotLowerPriorityToZeroOrBelow(t *testing.T) {
	cm, ctx := newManager(t)

	s, err := cm.FileSuggestion(ctx, catalog.Suggestion{Slug: "cat38-slug", Title: "Probe"})
	if err != nil {
		t.Fatalf("file: %v", err)
	}
	if s.Priority != 100 {
		t.Fatalf("filed priority = %d, want the declared default 100", s.Priority)
	}

	s, err = cm.DecideSuggestion(ctx, s.ID, catalog.SuggestionStatusApproved, 5)
	if err != nil {
		t.Fatalf("decide to 5: %v", err)
	}
	if s.Priority != 5 {
		t.Fatalf("priority after first decide = %d, want 5", s.Priority)
	}

	s, err = cm.DecideSuggestion(ctx, s.ID, catalog.SuggestionStatusApproved, 0)
	if err != nil {
		t.Fatalf("decide to 0: %v", err)
	}
	if s.Priority != 0 {
		t.Errorf("CAT38 open bug: DecideSuggestion(priority=0) left Priority = %d, want 0 — catalog.blueprint.json declares suggestion.priority as a plain int with no minimum (\"Queue position, lowest first\"), so zero is a legitimate, more-urgent-than-default position, but DecideSuggestion's `if priority > 0 { updates[\"priority\"] = priority }` (suggestion.go) silently treats zero as \"leave the stored value alone\" — decide_suggestion is the only declared operation that can change a filed suggestion's priority, so an operator can never move one to the front of the queue once it has held any positive priority", s.Priority)
	}

	s, err = cm.DecideSuggestion(ctx, s.ID, catalog.SuggestionStatusApproved, -10)
	if err != nil {
		t.Fatalf("decide to -10: %v", err)
	}
	if s.Priority != -10 {
		t.Errorf("CAT38 open bug: DecideSuggestion(priority=-10) left Priority = %d, want -10 — negative is a valid \"ahead of everything at the declared default of 100\" position under a lowest-first queue with no declared floor, but the same `priority > 0` guard silently drops it too", s.Priority)
	}
}
