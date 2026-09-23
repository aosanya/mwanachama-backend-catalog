package mwanachamacatalog_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func file(t *testing.T, cm catalog.CatalogManager, ctx context.Context, slug string) catalog.Suggestion {
	t.Helper()
	s, err := cm.FileSuggestion(ctx, catalog.Suggestion{
		Slug: slug, Title: "Build " + slug, Why: "worth having",
		Source: catalog.SuggestionSourceRun,
	})
	if err != nil {
		t.Fatalf("file %s: %v", slug, err)
	}
	return s
}

// approve moves a filed suggestion to approved, which is the only state a
// build attempt may be recorded against.
func approve(t *testing.T, cm catalog.CatalogManager, ctx context.Context, id string) catalog.Suggestion {
	t.Helper()
	s, err := cm.DecideSuggestion(ctx, id, catalog.SuggestionStatusApproved, 0)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	return s
}

func TestFileSuggestion_LandsProposedWithDefaults(t *testing.T) {
	cm, ctx := newManager(t)

	got := file(t, cm, ctx, "new-archetype")
	if got.Status != catalog.SuggestionStatusProposed {
		t.Errorf("Status = %q, want proposed", got.Status)
	}
	if got.Priority != 100 {
		t.Errorf("Priority = %d, want the default 100", got.Priority)
	}
	if got.Attempts != 0 || got.DecidedAt != "" {
		t.Errorf("got = %+v, want no attempts and no decision", got)
	}
}

// Dedup is not the caller's to skip. A slug that is already an agency, or
// already proposed, must be refused rather than filed twice.
func TestFileSuggestion_RefusesADuplicate(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "already-an-agency", catalog.VisibilityPublic)
	file(t, cm, ctx, "already-proposed")

	for _, slug := range []string{"already-an-agency", "already-proposed"} {
		_, err := cm.FileSuggestion(ctx, catalog.Suggestion{
			Slug: slug, Title: "Again", Source: catalog.SuggestionSourceRun,
		})
		if !errors.Is(err, catalog.ErrSuggestionDuplicate) {
			t.Errorf("%s: err = %v, want ErrSuggestionDuplicate", slug, err)
		}
	}
}

func TestFileSuggestion_RejectsBadInput(t *testing.T) {
	cm, ctx := newManager(t)

	for _, tc := range []struct {
		name string
		s    catalog.Suggestion
	}{
		{"no slug", catalog.Suggestion{Title: "One"}},
		{"bad slug", catalog.Suggestion{Slug: "Not A Slug", Title: "One"}},
		{"blank title", catalog.Suggestion{Slug: "one-two", Title: "   "}},
		{"unknown source", catalog.Suggestion{Slug: "one-two", Title: "One", Source: "somewhere"}},
		{"proposal is not json", catalog.Suggestion{Slug: "one-two", Title: "One", Proposal: "not json"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := cm.FileSuggestion(ctx, tc.s); !errors.Is(err, catalog.ErrInvalidSuggestion) {
				t.Fatalf("err = %v, want ErrInvalidSuggestion", err)
			}
		})
	}
}

// The tenth failed build stalls the proposal, so a run cannot retry
// something unbuildable forever.
func TestRecordSuggestionAttempt_StallsAtTheCeiling(t *testing.T) {
	cm, ctx := newManager(t)
	s := approve(t, cm, ctx, file(t, cm, ctx, "hard-one").ID)

	for i := 1; i < catalog.MaxSuggestionAttempts; i++ {
		got, err := cm.RecordSuggestionAttempt(ctx, s.ID, "run-1", fmt.Sprintf("attempt %d left work", i))
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		if got.Status != catalog.SuggestionStatusApproved {
			t.Fatalf("attempt %d: Status = %q, want still approved", i, got.Status)
		}
		if got.Attempts != i {
			t.Fatalf("attempt %d: Attempts = %d", i, got.Attempts)
		}
	}

	last, err := cm.RecordSuggestionAttempt(ctx, s.ID, "run-1", "still not done")
	if err != nil {
		t.Fatalf("last attempt: %v", err)
	}
	if last.Status != catalog.SuggestionStatusStalled {
		t.Errorf("Status = %q, want stalled at %d attempts", last.Status, catalog.MaxSuggestionAttempts)
	}

	// Once stalled it is no longer approved, so nothing may keep trying.
	if _, err := cm.RecordSuggestionAttempt(ctx, s.ID, "run-1", "one more"); !errors.Is(err, catalog.ErrSuggestionStatus) {
		t.Errorf("err = %v, want ErrSuggestionStatus once stalled", err)
	}
}

// Sending a stalled proposal back to approved is what gives it another ten
// tries; leaving the count would stall it again on the next attempt.
func TestDecideSuggestion_ApprovingClearsTheAttempts(t *testing.T) {
	cm, ctx := newManager(t)
	s := approve(t, cm, ctx, file(t, cm, ctx, "hard-one").ID)

	for i := 0; i < catalog.MaxSuggestionAttempts; i++ {
		if _, err := cm.RecordSuggestionAttempt(ctx, s.ID, "run-1", "left work"); err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	back := approve(t, cm, ctx, s.ID)
	if back.Attempts != 0 {
		t.Errorf("Attempts = %d, want cleared on approval", back.Attempts)
	}
	if back.Status != catalog.SuggestionStatusApproved {
		t.Errorf("Status = %q, want approved", back.Status)
	}
}

// Each attempt must say what is left, or the count rises toward the stall
// with no record of why.
func TestRecordSuggestionAttempt_NeedsAComment(t *testing.T) {
	cm, ctx := newManager(t)
	s := approve(t, cm, ctx, file(t, cm, ctx, "hard-one").ID)

	if _, err := cm.RecordSuggestionAttempt(ctx, s.ID, "run-1", "   "); !errors.Is(err, catalog.ErrAttemptNeedsComment) {
		t.Fatalf("err = %v, want ErrAttemptNeedsComment", err)
	}
	got, err := cm.GetSuggestion(ctx, s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Attempts != 0 {
		t.Errorf("Attempts = %d, want the refused attempt not counted", got.Attempts)
	}
}

func TestRecordSuggestionAttempt_KeepsTheReason(t *testing.T) {
	cm, ctx := newManager(t)
	s := approve(t, cm, ctx, file(t, cm, ctx, "hard-one").ID)

	if _, err := cm.RecordSuggestionAttempt(ctx, s.ID, "run-7", "the objectives are missing"); err != nil {
		t.Fatalf("attempt: %v", err)
	}
	comments, err := cm.ListSuggestionComments(ctx, s.ID)
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("len = %d, want 1", len(comments))
	}
	c := comments[0]
	if c.Author != catalog.CommentAuthorRun || c.Attempt != 1 || c.RunID != "run-7" {
		t.Errorf("comment = %+v, want a run comment for attempt 1", c)
	}
	if c.Body != "the objectives are missing" {
		t.Errorf("Body = %q", c.Body)
	}
}

// Only a build may mark a proposal built — deciding it so would claim an
// archetype exists that nothing built.
func TestDecideSuggestion_CannotSetBuilt(t *testing.T) {
	cm, ctx := newManager(t)
	s := file(t, cm, ctx, "new-one")

	if _, err := cm.DecideSuggestion(ctx, s.ID, catalog.SuggestionStatusBuilt, 0); !errors.Is(err, catalog.ErrSuggestionStatus) {
		t.Fatalf("err = %v, want ErrSuggestionStatus", err)
	}
}

func TestFulfilSuggestion_MarksItBuiltAndLinksTheEntry(t *testing.T) {
	cm, ctx := newManager(t)
	s := approve(t, cm, ctx, file(t, cm, ctx, "new-one").ID)
	seed(t, cm, ctx, "the-answer", catalog.VisibilityPending)

	got, err := cm.FulfilSuggestion(ctx, s.ID, "the-answer", "run-3", "built from the sketch")
	if err != nil {
		t.Fatalf("fulfil: %v", err)
	}
	if got.Status != catalog.SuggestionStatusBuilt {
		t.Errorf("Status = %q, want built", got.Status)
	}

	matches, err := cm.ListSuggestionMatches(ctx, s.ID)
	if err != nil {
		t.Fatalf("matches: %v", err)
	}
	if len(matches) != 1 || matches[0].Slug != "the-answer" {
		t.Fatalf("matches = %+v, want the-answer", matches)
	}

	// Fulfilling twice updates the match rather than filing a second.
	if _, err := cm.FulfilSuggestion(ctx, s.ID, "the-answer", "run-4", "again"); err != nil {
		t.Fatalf("fulfil again: %v", err)
	}
	matches, err = cm.ListSuggestionMatches(ctx, s.ID)
	if err != nil {
		t.Fatalf("matches again: %v", err)
	}
	if len(matches) != 1 || matches[0].RunID != "run-4" {
		t.Errorf("matches = %+v, want one match updated to run-4", matches)
	}
}

// A proposal cannot be marked built by pointing it at an agency that is not
// there.
func TestFulfilSuggestion_NeedsAnExistingEntry(t *testing.T) {
	cm, ctx := newManager(t)
	s := approve(t, cm, ctx, file(t, cm, ctx, "new-one").ID)

	if _, err := cm.FulfilSuggestion(ctx, s.ID, "no-such-agency", "", ""); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("err = %v, want ErrEntryNotFound", err)
	}
	got, err := cm.GetSuggestion(ctx, s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status == catalog.SuggestionStatusBuilt {
		t.Error("the suggestion was marked built with nothing answering it")
	}
}

func TestFulfilSuggestion_OnlyFromApproved(t *testing.T) {
	cm, ctx := newManager(t)
	s := file(t, cm, ctx, "new-one")
	seed(t, cm, ctx, "the-answer", catalog.VisibilityPending)

	if _, err := cm.FulfilSuggestion(ctx, s.ID, "the-answer", "", ""); !errors.Is(err, catalog.ErrSuggestionStatus) {
		t.Fatalf("err = %v, want ErrSuggestionStatus while proposed", err)
	}
}

func TestListSuggestions_ByStatusAndPriority(t *testing.T) {
	cm, ctx := newManager(t)
	a := file(t, cm, ctx, "first-one")
	b := file(t, cm, ctx, "second-one")
	file(t, cm, ctx, "third-one")

	if _, err := cm.DecideSuggestion(ctx, b.ID, catalog.SuggestionStatusApproved, 1); err != nil {
		t.Fatalf("approve b: %v", err)
	}
	if _, err := cm.DecideSuggestion(ctx, a.ID, catalog.SuggestionStatusApproved, 50); err != nil {
		t.Fatalf("approve a: %v", err)
	}

	got, err := cm.ListSuggestions(ctx, catalog.SuggestionStatusApproved)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 approved", len(got))
	}
	if got[0].Slug != "second-one" || got[1].Slug != "first-one" {
		t.Errorf("order = %q, %q; want best priority first", got[0].Slug, got[1].Slug)
	}

	all, err := cm.ListSuggestions(ctx)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("len = %d, want all 3", len(all))
	}
}
