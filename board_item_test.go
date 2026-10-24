package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestABoardItemCountsItsGroupedItemsAndItsMeToos(t *testing.T) {
	cm, ctx := newManager(t)
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{ID: "chosen", Title: "Streetlights out on the main road"})
	if err != nil {
		t.Fatal(err)
	}
	if b.ID == "" || b.ID == "chosen" || b.CreatedAt == "" {
		t.Fatalf("store-owned fields not minted: %+v", b)
	}
	for _, who := range []string{"amina", "baraka"} {
		r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Dark road", Publishable: true, RaisedBy: who})
		if err != nil {
			t.Fatal(err)
		}
		if err := cm.GroupUnderBoardItem(ctx, r.ID, b.ID); err != nil {
			t.Fatal(err)
		}
	}
	for _, who := range []string{"chiku", "chiku", "dawit"} {
		if _, err := cm.Endorse(ctx, b.ID, who); err != nil {
			t.Fatal(err)
		}
	}
	empty, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Market drainage", Outcome: "Drains cleared in May"})
	if err != nil {
		t.Fatal(err)
	}
	board, err := cm.ListBoardItems(ctx)
	if err != nil || len(board) != 2 {
		t.Fatalf("board = %+v (%v)", board, err)
	}
	byID := map[string]catalog.BoardItemTally{}
	for _, item := range board {
		byID[item.ID] = item
	}
	if got := byID[b.ID]; got.RaisedBy != 2 || got.MeToo != 2 {
		t.Errorf("grouped = %+v", got)
	}
	if got := byID[empty.ID]; got.RaisedBy != 0 || got.MeToo != 0 || got.Outcome != "Drains cleared in May" {
		t.Errorf("empty = %+v", got)
	}
}

func TestOnlyAPublishableItemIsGroupedUnderABoardItem(t *testing.T) {
	cm, ctx := newManager(t)
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Potholes"})
	if err != nil {
		t.Fatal(err)
	}
	private, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Pothole by my gate", RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, private.ID, b.ID); !errors.Is(err, catalog.ErrRaisedItemNotPublishable) {
		t.Errorf("private grouped: %v", err)
	}
	if err := cm.GroupUnderBoardItem(ctx, "nope", b.ID); !errors.Is(err, catalog.ErrRaisedItemNotFound) {
		t.Errorf("unknown item: %v", err)
	}
	if err := cm.GroupUnderBoardItem(ctx, private.ID, "nope"); !errors.Is(err, catalog.ErrBoardItemNotFound) {
		t.Errorf("unknown board item: %v", err)
	}
	if _, err := cm.PutOnBoard(ctx, catalog.BoardItem{}); !errors.Is(err, catalog.ErrInvalidBoardItem) {
		t.Errorf("no title: %v", err)
	}
	board, err := cm.ListBoardItems(ctx)
	if err != nil || len(board) != 1 || board[0].RaisedBy != 0 {
		t.Fatalf("board = %+v (%v)", board, err)
	}
	if public, err := cm.ListPublicBoardItems(ctx, 0, 0); err != nil || len(public) != 0 {
		t.Errorf("a board item with only an unpublishable item reached the public board: %+v (%v)", public, err)
	}
}

func TestABoardItemTakesItsPublicOutcomeAndLetsAnItemGo(t *testing.T) {
	cm, ctx := newManager(t)
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Broken bridge rail"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Rail", Publishable: true, RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, r.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if err := cm.SetBoardItemOutcome(ctx, b.ID, "Rail repaired"); err != nil {
		t.Fatal(err)
	}
	if err := cm.UngroupFromBoardItem(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	board, err := cm.ListBoardItems(ctx)
	if err != nil || len(board) != 1 {
		t.Fatalf("board = %+v (%v)", board, err)
	}
	if got := board[0]; got.Outcome != "Rail repaired" || got.RaisedBy != 0 {
		t.Errorf("after outcome and ungrouping = %+v", got)
	}
	if public, err := cm.ListPublicBoardItems(ctx, 0, 0); err != nil || len(public) != 0 {
		t.Errorf("an emptied board item stayed on the public board: %+v (%v)", public, err)
	}
	mine, err := cm.ListRaisedItems(ctx, "amina")
	if err != nil || len(mine) != 1 || mine[0].BoardItemID != "" {
		t.Fatalf("raised = %+v (%v)", mine, err)
	}
	if err := cm.SetBoardItemOutcome(ctx, "nobody", "x"); !errors.Is(err, catalog.ErrBoardItemNotFound) {
		t.Errorf("unknown board item = %v", err)
	}
	if err := cm.UngroupFromBoardItem(ctx, "nobody"); !errors.Is(err, catalog.ErrRaisedItemNotFound) {
		t.Errorf("unknown raised item = %v", err)
	}
}

func TestThePublicBoardLeavesOutABoardItemWithNothingGroupedUnderIt(t *testing.T) {
	cm, ctx := newManager(t)
	grouped, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Street lights"})
	if err != nil {
		t.Fatal(err)
	}
	empty, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Market drainage"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Dark road", Publishable: true, RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, r.ID, grouped.ID); err != nil {
		t.Fatal(err)
	}
	staff, err := cm.ListBoardItems(ctx)
	if err != nil || len(staff) != 2 {
		t.Fatalf("staff board = %+v (%v)", staff, err)
	}
	public, err := cm.ListPublicBoardItems(ctx, 0, 0)
	if err != nil || len(public) != 1 || public[0].ID != grouped.ID {
		t.Fatalf("public board = %+v (%v), want only %s and never %s", public, err, grouped.ID, empty.ID)
	}
	if err := cm.UngroupFromBoardItem(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if public, err = cm.ListPublicBoardItems(ctx, 0, 0); err != nil || len(public) != 0 {
		t.Fatalf("public board after ungrouping = %+v (%v)", public, err)
	}
}
