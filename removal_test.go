package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestARemovalNeedsAGroundAndShowsItToTheRaiserAlone(t *testing.T) {
	cm, ctx := newManager(t)
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Neighbour's phone number", RaisedBy: "amina", Publishable: true})
	if err != nil {
		t.Fatal(err)
	}
	other, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Streetlight out", RaisedBy: "baraka", Publishable: true})
	if err != nil {
		t.Fatal(err)
	}
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Things raised"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, r.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	for _, ground := range []string{"", "   "} {
		if _, err := cm.RemoveRaisedItem(ctx, catalog.Removal{RaisedItemID: r.ID, Ground: ground, RemovedBy: "clerk-1"}); !errors.Is(err, catalog.ErrInvalidRemoval) {
			t.Errorf("ground %q = %v", ground, err)
		}
	}
	if _, err := cm.RemoveRaisedItem(ctx, catalog.Removal{RaisedItemID: r.ID, Ground: "Names a private person"}); !errors.Is(err, catalog.ErrInvalidRemoval) {
		t.Errorf("no remover = %v", err)
	}
	if _, err := cm.RemoveRaisedItem(ctx, catalog.Removal{RaisedItemID: "nobody", Ground: "x", RemovedBy: "clerk-1"}); !errors.Is(err, catalog.ErrRaisedItemNotFound) {
		t.Errorf("unknown raised item = %v", err)
	}
	got, err := cm.RemoveRaisedItem(ctx, catalog.Removal{ID: "chosen", CreatedAt: "2000-01-01T00:00:00Z", RaisedItemID: r.ID, Ground: "Names a private person", RemovedBy: "clerk-1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == "" || got.ID == "chosen" || got.CreatedAt == "" || got.CreatedAt == "2000-01-01T00:00:00Z" {
		t.Fatalf("store-owned fields not minted: %+v", got)
	}
	if _, err := cm.RemoveRaisedItem(ctx, catalog.Removal{RaisedItemID: r.ID, Ground: "again", RemovedBy: "clerk-2"}); !errors.Is(err, catalog.ErrRaisedItemAlreadyRemoved) {
		t.Errorf("second removal = %v", err)
	}
	tallies, err := cm.ListBoardItems(ctx)
	if err != nil || len(tallies) != 1 || tallies[0].RaisedBy != 0 {
		t.Errorf("removed item still counted on the board: %+v (%v)", tallies, err)
	}
	if public, err := cm.ListPublicBoardItems(ctx, 0, 0); err != nil || len(public) != 0 {
		t.Errorf("a board item emptied by a removal stayed public: %+v (%v)", public, err)
	}
	if err := cm.GroupUnderBoardItem(ctx, r.ID, b.ID); !errors.Is(err, catalog.ErrRaisedItemNotPublishable) {
		t.Errorf("removed item regrouped = %v", err)
	}
	mine, err := cm.ListRemovalsOfMine(ctx, "amina")
	if err != nil || len(mine) != 1 || mine[0].Ground != "Names a private person" || mine[0].RemovedBy != "clerk-1" || mine[0].RaisedItemID != r.ID {
		t.Errorf("raiser's removals = %+v (%v)", mine, err)
	}
	theirs, err := cm.ListRemovalsOfMine(ctx, "baraka")
	if err != nil || len(theirs) != 0 {
		t.Errorf("another raiser sees %+v (%v)", theirs, err)
	}
	if _, err := cm.ListRemovalsOfMine(ctx, ""); !errors.Is(err, catalog.ErrInvalidRemoval) {
		t.Errorf("no raiser = %v", err)
	}
	_ = other
}
