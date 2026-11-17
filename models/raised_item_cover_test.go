package models

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/geo"
)

func TestRaisedItemOnBehalfIsWholeNeedsFiledByAndChannelTogether(t *testing.T) {
	cases := []struct {
		item RaisedItem
		want bool
	}{
		{RaisedItem{}, true},
		{RaisedItem{FiledBy: "agent", Channel: "phone"}, true},
		{RaisedItem{FiledBy: "agent"}, false},
		{RaisedItem{Channel: "phone"}, false},
	}
	for _, c := range cases {
		if got := c.item.OnBehalfIsWhole(); got != c.want {
			t.Fatalf("%+v = %v, want %v", c.item, got, c.want)
		}
	}
}

func TestRaisedItemPinIsAPointAcceptsNoPinAndPointsOnly(t *testing.T) {
	point := geo.NewPoint(36.8, -1.3)
	polygon := geo.Geometry{Kind: geo.KindPolygon}
	if !(RaisedItem{}).PinIsAPoint() {
		t.Fatal("no pin should pass")
	}
	if !(RaisedItem{Pin: &point}).PinIsAPoint() {
		t.Fatal("a point should pass")
	}
	if (RaisedItem{Pin: &polygon}).PinIsAPoint() {
		t.Fatal("a polygon should be refused")
	}
}

func TestRaisedItemAsSeenByOthersHidesAddressAndBlursThePinWithoutTouchingTheOriginal(t *testing.T) {
	pin := geo.NewPoint(36.817223456, -1.286389123)
	item := RaisedItem{ID: "r1", Address: "12 Kenyatta Ave", Pin: &pin, Area: "Westlands"}
	seen := item.AsSeenByOthers()
	if seen.Address != "" || seen.ID != "r1" || seen.Area != "Westlands" {
		t.Fatalf("seen = %+v", seen)
	}
	want := geo.RoundForPublic(pin.Point)
	if seen.Pin == nil || seen.Pin.Point != want || seen.Pin.Kind != geo.KindPoint {
		t.Fatalf("pin = %+v, want %v", seen.Pin, want)
	}
	if item.Address != "12 Kenyatta Ave" || item.Pin.Point != pin.Point {
		t.Fatalf("original changed: %+v", item)
	}
}

func TestRaisedItemAsSeenByOthersLeavesAnUnpinnedItemUnpinned(t *testing.T) {
	seen := (RaisedItem{Address: "somewhere"}).AsSeenByOthers()
	if seen.Pin != nil || seen.Address != "" {
		t.Fatalf("seen = %+v", seen)
	}
}

func TestRaisedItemNeverSerialisesItsFollowKeyHash(t *testing.T) {
	raw, err := json.Marshal(RaisedItem{ID: "r1", FollowKeyHash: "secret-hash"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret-hash") || strings.Contains(string(raw), "FollowKeyHash") {
		t.Fatalf("leaked: %s", raw)
	}
}
