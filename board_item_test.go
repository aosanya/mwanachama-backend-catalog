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
}
