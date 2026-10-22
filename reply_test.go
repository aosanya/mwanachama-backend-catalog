package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestARaisedItemTakesReplies(t *testing.T) {
	cm, ctx := newManager(t)
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Bin not collected", RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := cm.ReplyToRaisedItem(ctx, catalog.Reply{ID: "chosen", RaisedItemID: r.ID, Body: "Seen, a crew is booked", RepliedBy: "clerk-1"})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == "" || first.ID == "chosen" || first.CreatedAt == "" {
		t.Fatalf("store-owned fields not minted: %+v", first)
	}
	if _, err := cm.ReplyToRaisedItem(ctx, catalog.Reply{RaisedItemID: r.ID, Body: "Collected today", RepliedBy: "clerk-2"}); err != nil {
		t.Fatal(err)
	}
	got, err := cm.ListReplies(ctx, r.ID)
	if err != nil || len(got) != 2 {
		t.Fatalf("replies = %+v (%v)", got, err)
	}
	bodies := map[string]bool{got[0].Body: true, got[1].Body: true}
	if !bodies["Seen, a crew is booked"] || !bodies["Collected today"] {
		t.Errorf("replies = %+v", got)
	}
	if _, err := cm.ReplyToRaisedItem(ctx, catalog.Reply{RaisedItemID: "nobody", Body: "x", RepliedBy: "clerk-1"}); !errors.Is(err, catalog.ErrRaisedItemNotFound) {
		t.Errorf("unknown raised item = %v", err)
	}
	if _, err := cm.ReplyToRaisedItem(ctx, catalog.Reply{RaisedItemID: r.ID, RepliedBy: "clerk-1"}); !errors.Is(err, catalog.ErrInvalidReply) {
		t.Errorf("empty body = %v", err)
	}
}
