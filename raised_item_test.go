package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestARaisedItemIsListedForItsRaiserAlone(t *testing.T) {
	cm, ctx := newManager(t)
	mine, err := cm.RaiseItem(ctx, catalog.RaisedItem{ID: "chosen", Title: "Broken streetlight", Topic: "lighting", Publishable: true, RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	if mine.ID == "" || mine.ID == "chosen" || mine.CreatedAt == "" {
		t.Fatalf("store-owned fields not minted: %+v", mine)
	}
	if _, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Market drainage", RaisedBy: "baraka", FiledBy: "clerk-1", Channel: "counter"}); err != nil {
		t.Fatal(err)
	}
	listed, err := cm.ListRaisedItems(ctx, "amina")
	if err != nil || len(listed) != 1 || listed[0].ID != mine.ID || !listed[0].Publishable || listed[0].Topic != "lighting" {
		t.Fatalf("amina's items = %+v (%v)", listed, err)
	}
	theirs, err := cm.ListRaisedItems(ctx, "baraka")
	if err != nil || len(theirs) != 1 || theirs[0].FiledBy != "clerk-1" || theirs[0].Channel != "counter" || theirs[0].Publishable {
		t.Fatalf("baraka's items = %+v (%v)", theirs, err)
	}
}

func TestARaisedItemNeedsATitleARaiserAndAWholeOnBehalf(t *testing.T) {
	cm, ctx := newManager(t)
	for name, r := range map[string]catalog.RaisedItem{
		"no title":         {RaisedBy: "amina"},
		"no raiser":        {Title: "Potholes"},
		"filer no channel": {Title: "Potholes", RaisedBy: "amina", FiledBy: "clerk-1"},
		"channel no filer": {Title: "Potholes", RaisedBy: "amina", Channel: "phone"},
	} {
		if _, err := cm.RaiseItem(ctx, r); !errors.Is(err, catalog.ErrInvalidRaisedItem) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := cm.ListRaisedItems(ctx, ""); !errors.Is(err, catalog.ErrInvalidRaisedItem) {
		t.Errorf("no raiser listed: %v", err)
	}
}
