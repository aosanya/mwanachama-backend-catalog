package mwanachamacatalog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func withExtraSuggestionStatus(t *testing.T, extra string) *spec.Spec {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(blueprintSource()), &doc); err != nil {
		t.Fatalf("read blueprint: %v", err)
	}
	widened := false
	for _, o := range doc["objects"].([]any) {
		obj := o.(map[string]any)
		if obj["role"] != roleSuggestion {
			continue
		}
		for _, f := range obj["fields"].([]any) {
			field := f.(map[string]any)
			if field["name"] == "status" {
				field["values"] = append(field["values"].([]any), extra)
				widened = true
			}
		}
	}
	if !widened {
		t.Fatalf("precondition: the blueprint's suggestion role declares no status field to widen")
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("rewrite blueprint: %v", err)
	}
	s, err := loadThrough(t, string(raw))
	if err != nil {
		t.Fatalf("load the example through the mutated blueprint: %v", err)
	}
	return s
}

func TestWidenedSuggestionStatusOpensNoDecisionGate(t *testing.T) {
	const fasttracked = models.SuggestionStatus("fasttracked")
	cm := managerOn(t, withExtraSuggestionStatus(t, string(fasttracked)))
	ctx := context.Background()

	s, err := cm.FileSuggestion(ctx, Suggestion{
		Slug: "one-suggestion", Title: "Build one", Why: "worth having",
		Source: models.SuggestionSourceRun,
	})
	if err != nil || s.Status != models.SuggestionStatusProposed {
		t.Fatalf("precondition: filing lands a suggestion in proposed, got %q, err %v", s.Status, err)
	}

	if _, err := cm.DecideSuggestion(ctx, s.ID, fasttracked, 0); !errors.Is(err, ErrSuggestionStatus) {
		t.Errorf("a decision into domain-declared status %q was accepted: err = %v, want ErrSuggestionStatus", fasttracked, err)
	}
	if _, err := cm.DecideSuggestion(ctx, s.ID, models.SuggestionStatusBuilt, 0); !errors.Is(err, ErrSuggestionStatus) {
		t.Errorf("a decision straight to built was accepted: err = %v, want ErrSuggestionStatus", err)
	}

	got, err := cm.GetSuggestion(ctx, s.ID)
	if err != nil || got.Status != models.SuggestionStatusProposed {
		t.Fatalf("a refused decision moved the suggestion: status %q, err %v", got.Status, err)
	}

	if _, err := cm.RecordSuggestionAttempt(ctx, s.ID, "run-1", "tried"); !errors.Is(err, ErrSuggestionStatus) {
		t.Errorf("a build attempt was recorded against a suggestion that was never approved: err = %v, want ErrSuggestionStatus", err)
	}
	if _, err := cm.FulfilSuggestion(ctx, s.ID, "one-entry", "run-1", "built"); !errors.Is(err, ErrSuggestionStatus) {
		t.Errorf("a suggestion that was never approved was fulfilled: err = %v, want ErrSuggestionStatus", err)
	}
}
