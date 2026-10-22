package mwanachamacatalog_test

import (
	"errors"
	"reflect"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestASharedBoardItemOpensWithItsTitleThemeMeTooAndStatus(t *testing.T) {
	cm, ctx := newManager(t)
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Streetlights out", Outcome: "Reported to the utility"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Dark road", Topic: "lighting", Publishable: true, RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, r.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	for _, who := range []string{"baraka", "chiku"} {
		if _, err := cm.Endorse(ctx, b.ID, who); err != nil {
			t.Fatal(err)
		}
	}
	got, err := cm.OpenSharedBoardItem(ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := catalog.SharedBoardItem{Title: "Streetlights out", Theme: "lighting", MeToo: 2, Status: "Reported to the utility", Files: []string{}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shared = %+v, want %+v", got, want)
	}
}

func TestEveryMissOnASharedBoardItemAnswersTheSame(t *testing.T) {
	cm, ctx := newManager(t)
	unpublished, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Nothing grouped yet"})
	if err != nil {
		t.Fatal(err)
	}
	ungrouped, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Grouped then taken off"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Pothole", Publishable: true, RaisedBy: "amina"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, r.ID, ungrouped.ID); err != nil {
		t.Fatal(err)
	}
	if err := cm.UngroupFromBoardItem(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"", "no-such-key", unpublished.ID, ungrouped.ID} {
		got, err := cm.OpenSharedBoardItem(ctx, key)
		if !errors.Is(err, catalog.ErrBoardItemNotFound) || err.Error() != catalog.ErrBoardItemNotFound.Error() || !reflect.DeepEqual(got, catalog.SharedBoardItem{}) {
			t.Errorf("key %q: %+v, %v", key, got, err)
		}
	}
}
