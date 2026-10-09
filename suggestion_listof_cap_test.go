package mwanachamacatalog_test

import (
	"fmt"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT45_OpenBug_ListSuggestionsHasNoCapAndNoWayToAskForOne(t *testing.T) {
	cm, ctx := newManager(t)

	const filed = 300
	for i := 0; i < filed; i++ {
		if _, err := cm.FileSuggestion(ctx, catalog.Suggestion{
			Slug:  fmt.Sprintf("probe-%03d", i),
			Title: "probe",
		}); err != nil {
			t.Fatalf("file %d: %v", i, err)
		}
	}

	got, err := cm.ListSuggestions(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) >= filed {
		t.Fatalf("CAT45 open bug: ListSuggestions() returned all %d of %d filed suggestions, and its signature takes no limit parameter at all, so no caller could cap this call even if it wanted to; a caller who names no limit should get a capped default page, not every suggestion ever filed — including the anonymously-filed ones CAT29 already shows are unbounded to create", len(got), filed)
	}
}
