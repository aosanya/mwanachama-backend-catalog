package mwanachamacatalog_test

import (
	"errors"
	"testing"
	"time"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestOverdueRaisedItemsAreTheOpenOnesPastTheAnswerClock(t *testing.T) {
	cm, ctx := newManager(t)
	open, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Bin not collected", RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	answered, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Streetlight out", RaisedBy: "baraka"})
	if err != nil {
		t.Fatal(err)
	}
	removed, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Names a neighbour", RaisedBy: "chausiku"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.ReplyToRaisedItem(ctx, catalog.Reply{RaisedItemID: answered.ID, Body: "Fixed", RepliedBy: "clerk-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.RemoveRaisedItem(ctx, catalog.Removal{RaisedItemID: removed.ID, Ground: "names a person", RemovedBy: "clerk-1"}); err != nil {
		t.Fatal(err)
	}

	if _, err := cm.ListOverdueRaisedItems(ctx, ""); !errors.Is(err, catalog.ErrSettingNotFound) {
		t.Fatalf("no clock set = %v, want ErrSettingNotFound", err)
	}
	for _, bad := range []string{"ten", "0", "-3"} {
		if _, err := cm.PutSetting(ctx, catalog.AnswerClockSetting, bad); err != nil {
			t.Fatal(err)
		}
		if _, err := cm.ListOverdueRaisedItems(ctx, ""); !errors.Is(err, catalog.ErrInvalidSetting) {
			t.Errorf("clock %q = %v, want ErrInvalidSetting", bad, err)
		}
	}
	if _, err := cm.PutSetting(ctx, catalog.AnswerClockSetting, "10"); err != nil {
		t.Fatal(err)
	}

	raised, err := time.Parse(time.RFC3339, open.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	at := func(days int) string { return raised.Add(time.Duration(days) * 24 * time.Hour).Format(time.RFC3339) }

	if got, err := cm.ListOverdueRaisedItems(ctx, at(9)); err != nil || len(got) != 0 {
		t.Errorf("nine days in = %+v (%v), want nothing overdue", got, err)
	}
	got, err := cm.ListOverdueRaisedItems(ctx, at(11))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != open.ID {
		t.Errorf("eleven days in = %+v, want only the unanswered, unremoved item", got)
	}
	if _, err := cm.ListOverdueRaisedItems(ctx, "yesterday"); !errors.Is(err, catalog.ErrInvalidRaisedItem) {
		t.Errorf("unparseable time = %v, want ErrInvalidRaisedItem", err)
	}
}
