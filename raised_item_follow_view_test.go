package mwanachamacatalog_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAFollowerReadsStatusAndReplyTextAndNoneOfTheStaffWhoActed(t *testing.T) {
	cm, ctx := newManager(t)
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Bin not collected", RaisedBy: "amina-member-id", Address: "12 Elm Road"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.ReplyToRaisedItem(ctx, catalog.Reply{RaisedItemID: r.ID, Body: "A crew is booked", RepliedBy: "clerk-member-id"}); err != nil {
		t.Fatal(err)
	}
	key, err := cm.IssueFollowKey(ctx, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	view, err := cm.ShowFollowedRaisedItem(ctx, key.Key)
	if err != nil {
		t.Fatal(err)
	}
	if view.Item.Title != "Bin not collected" || view.Status == "" || len(view.Replies) != 1 || view.Replies[0].Body != "A crew is booked" {
		t.Fatalf("follower view = %+v", view)
	}
	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	for _, leaked := range []string{"amina-member-id", "clerk-member-id", "raised_by", "replied_by", "department_by"} {
		if strings.Contains(string(raw), leaked) {
			t.Errorf("the follower view carries %q: %s", leaked, raw)
		}
	}
}

func TestAMissedFollowKeyIsNotFoundInTheFollowerViewToo(t *testing.T) {
	cm, ctx := newManager(t)
	if _, err := cm.ShowFollowedRaisedItem(ctx, "no-such-key"); !errors.Is(err, catalog.ErrRaisedItemNotFound) {
		t.Fatalf("err = %v", err)
	}
}
