package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT47_DecideBuiltSuggestionIsRefusedAndTheMatchSurvives(t *testing.T) {
	cm, ctx := newManager(t)
	s := approve(t, cm, ctx, file(t, cm, ctx, "cat47-probe").ID)
	seed(t, cm, ctx, "cat47-entry", catalog.VisibilityPending)

	built, err := cm.FulfilSuggestion(ctx, s.ID, "cat47-entry", "run-1", "built it")
	if err != nil {
		t.Fatalf("fulfil: %v", err)
	}
	if built.Status != catalog.SuggestionStatusBuilt {
		t.Fatalf("Status = %q, want built", built.Status)
	}

	got, err := cm.DecideSuggestion(ctx, s.ID, catalog.SuggestionStatusRejected, nil)
	if !errors.Is(err, catalog.ErrSuggestionStatus) {
		t.Fatalf("CAT47: DecideSuggestion(rejected) on a built suggestion returned status %q, err %v; want ErrSuggestionStatus, because an entry already answers this suggestion and nothing should be able to disown it by deciding the proposal that built it", got.Status, err)
	}

	after, err := cm.GetSuggestion(ctx, s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if after.Status != catalog.SuggestionStatusBuilt {
		t.Fatalf("CAT47: suggestion read back as %q after the refused decide; want it to stay built", after.Status)
	}

	matches, err := cm.ListSuggestionMatches(ctx, s.ID)
	if err != nil {
		t.Fatalf("matches: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("CAT47: %d match rows remain for a suggestion the catalog now calls %q; a suggestion the catalog says nothing answered should carry no match", len(matches), after.Status)
	}
}
