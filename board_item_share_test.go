package mwanachamacatalog_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-shared/geo"
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

func TestASharedBoardItemCarriesTheAreaButNeverTheAddressPinOrRaiser(t *testing.T) {
	cm, ctx := newManager(t)
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Streetlights out"})
	if err != nil {
		t.Fatal(err)
	}
	pin := geo.NewPoint(36.817223, -1.286389)
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Dark road", Topic: "lighting", Publishable: true, RaisedBy: "amina", Pin: &pin, Address: "12 Kenyatta Avenue", Area: "Central"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, r.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	got, err := cm.OpenSharedBoardItem(ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Area != "Central" {
		t.Fatalf("area = %q, want Central", got.Area)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, leaked := range []string{"Kenyatta", "amina", "36.8", "1.28", r.ID} {
		if strings.Contains(string(raw), leaked) {
			t.Errorf("shared item %s carries %q", raw, leaked)
		}
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
