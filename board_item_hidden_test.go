package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAHiddenBoardItemIsLeftOutOfTheBoardAndItsShareLink(t *testing.T) {
	cm, ctx := newManager(t)
	shown, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Streetlights out"})
	if err != nil {
		t.Fatal(err)
	}
	hidden, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Rubbish at the market"})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range []catalog.BoardItem{shown, hidden} {
		r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: b.Title, Publishable: true, RaisedBy: "amina"})
		if err != nil {
			t.Fatal(err)
		}
		if err := cm.GroupUnderBoardItem(ctx, r.ID, b.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := cm.SetBoardItemHidden(ctx, hidden.ID, true); err != nil {
		t.Fatal(err)
	}
	board, err := cm.ListBoardItems(ctx)
	if err != nil || len(board) != 1 || board[0].ID != shown.ID {
		t.Fatalf("board = %+v (%v)", board, err)
	}
	if _, err := cm.OpenSharedBoardItem(ctx, hidden.ID); !errors.Is(err, catalog.ErrBoardItemNotFound) {
		t.Fatalf("hidden share link opened: %v", err)
	}
	if _, err := cm.OpenSharedBoardItem(ctx, shown.ID); err != nil {
		t.Fatalf("shown share link refused: %v", err)
	}
	if err := cm.SetBoardItemHidden(ctx, hidden.ID, false); err != nil {
		t.Fatal(err)
	}
	if board, err = cm.ListBoardItems(ctx); err != nil || len(board) != 2 {
		t.Fatalf("unhidden board = %+v (%v)", board, err)
	}
	if _, err := cm.OpenSharedBoardItem(ctx, hidden.ID); err != nil {
		t.Fatalf("unhidden share link refused: %v", err)
	}
}

func TestHidingAnUnknownBoardItemIsNotFound(t *testing.T) {
	cm, ctx := newManager(t)
	if err := cm.SetBoardItemHidden(ctx, "nope", true); !errors.Is(err, catalog.ErrBoardItemNotFound) {
		t.Fatalf("err = %v", err)
	}
	if err := cm.SetBoardItemHidden(ctx, "", true); !errors.Is(err, catalog.ErrInvalidBoardItem) {
		t.Fatalf("err = %v", err)
	}
}
