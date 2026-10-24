package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/models"
)

func boardItem(t *testing.T, cm catalog.CatalogManager, id string) catalog.BoardItemTally {
	t.Helper()
	board, err := cm.ListBoardItems(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range board {
		if b.ID == id {
			return b
		}
	}
	t.Fatalf("board item %s not on the board", id)
	return catalog.BoardItemTally{}
}

func TestABoardItemIsAssignedToAPersonOrARoleAndUnassigned(t *testing.T) {
	cm, ctx := newManager(t)
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Streetlights out"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.AssignBoardItem(ctx, b.ID, models.AssigneeActor, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	if got := boardItem(t, cm, b.ID); got.AssigneeKind != "actor" || got.AssigneeID != "clerk-1" {
		t.Fatalf("assignee = %q %q", got.AssigneeKind, got.AssigneeID)
	}
	if err := cm.AssignBoardItem(ctx, b.ID, models.AssigneeRole, "roads-desk"); err != nil {
		t.Fatal(err)
	}
	if got := boardItem(t, cm, b.ID); got.AssigneeKind != "role" || got.AssigneeID != "roads-desk" || got.ClosedAt != "" {
		t.Fatalf("reassigned = %+v", got)
	}
	if err := cm.UnassignBoardItem(ctx, b.ID); err != nil {
		t.Fatal(err)
	}
	if got := boardItem(t, cm, b.ID); got.AssigneeKind != "" || got.AssigneeID != "" {
		t.Fatalf("unassigned = %+v", got)
	}
}

func TestAssigningRefusesAnUnknownKindAnEmptyAssigneeAndAnUnknownItem(t *testing.T) {
	cm, ctx := newManager(t)
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Streetlights out"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.AssignBoardItem(ctx, b.ID, "agent", "bot"); !errors.Is(err, catalog.ErrInvalidBoardItem) {
		t.Fatalf("unknown kind: %v", err)
	}
	if err := cm.AssignBoardItem(ctx, b.ID, models.AssigneeActor, ""); !errors.Is(err, catalog.ErrInvalidBoardItem) {
		t.Fatalf("empty assignee: %v", err)
	}
	if err := cm.AssignBoardItem(ctx, "nope", models.AssigneeActor, "clerk-1"); !errors.Is(err, catalog.ErrBoardItemNotFound) {
		t.Fatalf("unknown item: %v", err)
	}
	if err := cm.UnassignBoardItem(ctx, "nope"); !errors.Is(err, catalog.ErrBoardItemNotFound) {
		t.Fatalf("unassign unknown: %v", err)
	}
}

func TestPuttingOnTheBoardIgnoresAClaimedAssigneeAndClose(t *testing.T) {
	cm, ctx := newManager(t)
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Streetlights out", AssigneeKind: "actor", AssigneeID: "x", ClosedAt: "2026-01-01T00:00:00Z", ClosedBy: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if got := boardItem(t, cm, b.ID); got.AssigneeKind != "" || got.AssigneeID != "" || got.ClosedAt != "" || got.ClosedBy != "" {
		t.Fatalf("put = %+v", got)
	}
}

func TestOnlyCloseClosesABoardItemAndOnlyOnce(t *testing.T) {
	cm, ctx := newManager(t)
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Streetlights out"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Dark road", Publishable: true, RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, r.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if err := cm.SetBoardItemOutcome(ctx, b.ID, "Fixed"); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.ReplyToRaisedItem(ctx, catalog.Reply{RaisedItemID: r.ID, Body: "On it", RepliedBy: "clerk-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.RemoveRaisedItem(ctx, catalog.Removal{RaisedItemID: r.ID, Ground: "Names a private person", RemovedBy: "clerk-1"}); err != nil {
		t.Fatal(err)
	}
	if got := boardItem(t, cm, b.ID); got.ClosedAt != "" {
		t.Fatalf("closed by a reply, a removal or an outcome: %+v", got)
	}
	if err := cm.CloseBoardItem(ctx, b.ID, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	got := boardItem(t, cm, b.ID)
	if got.ClosedAt == "" || got.ClosedBy != "clerk-1" {
		t.Fatalf("closed = %+v", got)
	}
	if err := cm.CloseBoardItem(ctx, b.ID, "clerk-2"); !errors.Is(err, catalog.ErrBoardItemClosed) {
		t.Fatalf("second close: %v", err)
	}
	if again := boardItem(t, cm, b.ID); again.ClosedBy != "clerk-1" || again.ClosedAt != got.ClosedAt {
		t.Fatalf("second close moved it: %+v", again)
	}
	if err := cm.CloseBoardItem(ctx, "nope", "clerk-1"); !errors.Is(err, catalog.ErrBoardItemNotFound) {
		t.Fatalf("unknown item: %v", err)
	}
	if err := cm.CloseBoardItem(ctx, b.ID, ""); !errors.Is(err, catalog.ErrInvalidBoardItem) {
		t.Fatalf("no closer: %v", err)
	}
}
