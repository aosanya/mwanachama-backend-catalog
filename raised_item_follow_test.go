package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAFollowKeyOpensOneRaisedItemAndItsRepliesAndEveryMissIsNotFound(t *testing.T) {
	cm, ctx := newManager(t)
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Bin not collected", RaisedBy: "amina", Address: "12 Elm Road"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Streetlight out", RaisedBy: "baraka"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.ReplyToRaisedItem(ctx, catalog.Reply{RaisedItemID: r.ID, Body: "A crew is booked", RepliedBy: "clerk-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.ReplyToRaisedItem(ctx, catalog.Reply{RaisedItemID: other.ID, Body: "Not yours", RepliedBy: "clerk-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.OpenFollowedRaisedItem(ctx, ""); !errors.Is(err, catalog.ErrRaisedItemNotFound) {
		t.Errorf("empty key before any issue = %v", err)
	}
	first, err := cm.IssueFollowKey(ctx, r.ID)
	if err != nil || len(first.Key) != 64 || first.RaisedItemID != r.ID {
		t.Fatalf("issue = %+v (%v)", first, err)
	}
	got, err := cm.OpenFollowedRaisedItem(ctx, first.Key)
	if err != nil {
		t.Fatal(err)
	}
	if got.Item.ID != r.ID || got.Item.Address != "12 Elm Road" || len(got.Replies) != 1 || got.Replies[0].Body != "A crew is booked" {
		t.Fatalf("followed = %+v", got)
	}
	second, err := cm.IssueFollowKey(ctx, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	for name, key := range map[string]string{"retired": first.Key, "unknown": "deadbeef", "empty": ""} {
		if _, err := cm.OpenFollowedRaisedItem(ctx, key); !errors.Is(err, catalog.ErrRaisedItemNotFound) {
			t.Errorf("%s key = %v", name, err)
		}
	}
	if _, err := cm.OpenFollowedRaisedItem(ctx, second.Key); err != nil {
		t.Errorf("current key = %v", err)
	}
	if _, err := cm.IssueFollowKey(ctx, "nobody"); !errors.Is(err, catalog.ErrRaisedItemNotFound) {
		t.Errorf("missing raised item = %v", err)
	}
}
