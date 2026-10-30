package mwanachamacatalog_test

import (
	"encoding/json"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/models"
)

func TestCAT36_UpdateEntryKeepsAModeratorsApprovalWhenReviewedContentIsUnchanged(t *testing.T) {
	var ops struct {
		Operations map[string]struct {
			Call string `json:"call"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(catalog.Operations(), &ops); err != nil {
		t.Fatalf("parse catalog.operations.json: %v", err)
	}
	for name, op := range ops.Operations {
		if op.Call == "UpdateEntry" {
			t.Fatalf("precondition: operation %q already calls UpdateEntry; CAT36 assumed it reaches no operation", name)
		}
	}

	cm, ctx := newManager(t)
	before := seed(t, cm, ctx, "gamma", models.VisibilityPending)

	approved, err := cm.SetVisibility(ctx, "gamma", models.VisibilityPublic)
	if err != nil {
		t.Fatalf("precondition: SetVisibility(public): %v", err)
	}
	if approved.ApprovedAt == "" {
		t.Fatalf("precondition: a moderator's approval should set ApprovedAt")
	}

	_, err = cm.UpdateEntry(ctx, catalog.Entry{
		Slug:       "gamma",
		Name:       before.Name,
		Summary:    before.Summary,
		Doc:        before.Doc,
		DisposeAt:  "2099-01-01T00:00:00Z",
		Visibility: before.Visibility,
		ApprovedAt: before.ApprovedAt,
	})
	if err != nil {
		t.Fatalf("UpdateEntry with the pre-approval snapshot's own valid fields: %v", err)
	}

	stored, err := cm.GetEntry(ctx, "gamma")
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	_, openErr := cm.OpenEntry(ctx, "gamma", "")
	if stored.Visibility != models.VisibilityPublic || stored.ApprovedAt == "" || openErr != nil {
		t.Errorf("CAT36: UpdateEntry is reachable from no declared operation in catalog.operations.json — a caller that imports this module directly and calls it anyway, to revise an Entry's retention time, must already hold a full valid Entry (UpdateEntry requires a non-empty enum Visibility or refuses with ErrInvalidEntry) and so naturally resupplies whatever Visibility/ApprovedAt it read earlier; here that earlier read predates a moderator's SetVisibility(public), and the later UpdateEntry call writes the stale pre-approval visibility=%q, approved_at=%q straight back over the approval with no merge and no re-check — got stored visibility=%q approved_at=%q, OpenEntry(anon) err=%v; want visibility=%q, a non-empty ApprovedAt and OpenEntry to succeed, because revising a field outside the reviewed card must not silently undo a moderator's already-committed approval", before.Visibility, before.ApprovedAt, stored.Visibility, stored.ApprovedAt, openErr, models.VisibilityPublic)
	}
}
