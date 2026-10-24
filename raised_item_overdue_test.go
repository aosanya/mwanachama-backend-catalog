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

	raised, err := time.Parse(time.RFC3339, open.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	at := func(days int) string { return raised.Add(time.Duration(days) * 24 * time.Hour).Format(time.RFC3339) }

	if got, err := cm.ListOverdueRaisedItems(ctx, at(11)); err != nil || len(got) != 0 {
		t.Errorf("no clock set, eleven calendar days in = %+v (%v), want nothing: the default is ten weekdays", got, err)
	}
	if got, err := cm.ListOverdueRaisedItems(ctx, at(15)); err != nil || len(got) != 1 || got[0].ID != open.ID {
		t.Errorf("no clock set, fifteen calendar days in = %+v (%v), want only the open item", got, err)
	}
	for _, bad := range []string{"ten", "0", "-3"} {
		if _, err := cm.PutSetting(ctx, catalog.AnswerClockSetting, bad); err != nil {
			t.Fatal(err)
		}
		if _, err := cm.ListOverdueRaisedItems(ctx, ""); !errors.Is(err, catalog.ErrInvalidSetting) {
			t.Errorf("clock %q = %v, want ErrInvalidSetting", bad, err)
		}
	}
	if _, err := cm.PutSetting(ctx, catalog.AnswerClockSetting, "5"); err != nil {
		t.Fatal(err)
	}
	if got, err := cm.ListOverdueRaisedItems(ctx, at(4)); err != nil || len(got) != 0 {
		t.Errorf("four calendar days in, a five-weekday clock = %+v (%v), want nothing overdue", got, err)
	}
	got, err := cm.ListOverdueRaisedItems(ctx, at(9))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != open.ID {
		t.Errorf("nine calendar days in, a five-weekday clock = %+v, want only the unanswered, unremoved item", got)
	}
	if _, err := cm.ListOverdueRaisedItems(ctx, "yesterday"); !errors.Is(err, catalog.ErrInvalidRaisedItem) {
		t.Errorf("unparseable time = %v, want ErrInvalidRaisedItem", err)
	}
}

func TestOnlyAReplyToTheRaiserTakesAnItemOffTheOverdueList(t *testing.T) {
	cm, ctx := newManager(t)
	grouped, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Pothole on Market Street", Publishable: true, RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Potholes"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, grouped.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if err := cm.SetBoardItemOutcome(ctx, b.ID, "Resurfacing scheduled for May"); err != nil {
		t.Fatal(err)
	}
	raised, err := time.Parse(time.RFC3339, grouped.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	later := raised.Add(15 * 24 * time.Hour).Format(time.RFC3339)

	got, err := cm.ListOverdueRaisedItems(ctx, later)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != grouped.ID {
		t.Fatalf("grouped with an outcome = %+v, want it still overdue", got)
	}
	if _, err := cm.ReplyToRaisedItem(ctx, catalog.Reply{RaisedItemID: grouped.ID, Body: "On the May list", RepliedBy: "clerk-1"}); err != nil {
		t.Fatal(err)
	}
	if got, err := cm.ListOverdueRaisedItems(ctx, later); err != nil || len(got) != 0 {
		t.Errorf("after the reply = %+v (%v), want nothing overdue", got, err)
	}
}
