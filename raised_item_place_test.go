package mwanachamacatalog_test

import (
	"context"
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-shared/geo"
)

func pin(lon, lat float64) *geo.Geometry {
	g := geo.NewPoint(lon, lat)
	return &g
}

func TestTheAreaIsTheTypedPickElseThePinElseTheAddress(t *testing.T) {
	cm, ctx := newManager(t)
	var asked []string
	cm.UseAreaResolver(func(_ context.Context, at *geo.Position, address string) (string, error) {
		switch {
		case at != nil && at.Lat() > 0:
			asked = append(asked, "pin")
			return "north", nil
		case at != nil:
			asked = append(asked, "pin")
			return "", nil
		case address == "Riverside Road":
			asked = append(asked, "address")
			return "riverside", nil
		default:
			asked = append(asked, "address")
			return "", nil
		}
	})
	for name, c := range map[string]struct {
		item catalog.RaisedItem
		want string
	}{
		"pin wins over address":                   {catalog.RaisedItem{Pin: pin(36.8, 1.2), Address: "Riverside Road"}, "north"},
		"pin outside every area falls to address": {catalog.RaisedItem{Pin: pin(36.8, -1.2), Address: "Riverside Road"}, "riverside"},
		"a typed pick is kept over the pin":       {catalog.RaisedItem{Pin: pin(36.8, 1.2), Address: "Riverside Road", Area: "picked"}, "picked"},
		"unknown address falls to the pick":       {catalog.RaisedItem{Address: "Nowhere Lane", Area: "picked"}, "picked"},
		"nothing at all is no area":               {catalog.RaisedItem{}, ""},
	} {
		c.item.Title, c.item.RaisedBy = "Blocked drain", "amina"
		got, err := cm.RaiseItem(ctx, c.item)
		if err != nil || got.Area != c.want {
			t.Errorf("%s: area %q (%v), want %q", name, got.Area, err, c.want)
		}
	}
	if len(asked) == 0 {
		t.Fatal("the resolver was never asked")
	}
}

func TestWithoutAResolverThePickedAreaStands(t *testing.T) {
	cm, ctx := newManager(t)
	got, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Blocked drain", RaisedBy: "amina", Pin: pin(36.8, 1.2), Area: "picked"})
	if err != nil || got.Area != "picked" {
		t.Fatalf("area %q (%v)", got.Area, err)
	}
}

func TestAResolverFailureRefusesTheItem(t *testing.T) {
	cm, ctx := newManager(t)
	boom := errors.New("boom")
	cm.UseAreaResolver(func(context.Context, *geo.Position, string) (string, error) { return "", boom })
	if _, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "x", RaisedBy: "amina", Pin: pin(1, 1)}); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if listed, _ := cm.ListRaisedItems(ctx, "amina"); len(listed) != 0 {
		t.Fatalf("stored despite the failure: %+v", listed)
	}
}

func TestAPinMustBeAPoint(t *testing.T) {
	cm, ctx := newManager(t)
	shape := geo.NewPolygon(geo.Ring{{0, 0}, {1, 0}, {1, 1}, {0, 0}})
	if _, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "x", RaisedBy: "amina", Pin: &shape}); !errors.Is(err, catalog.ErrInvalidRaisedItem) {
		t.Fatalf("err = %v", err)
	}
}

func TestTheRaiserReadsTheExactPinAndOthersReadTheAreaRounded(t *testing.T) {
	cm, ctx := newManager(t)
	if _, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Pothole", RaisedBy: "amina", Pin: pin(36.821947, -1.292066), Address: "12 Riverside Road", Area: "riverside"}); err != nil {
		t.Fatal(err)
	}
	mine, err := cm.ListRaisedItems(ctx, "amina")
	if err != nil || len(mine) != 1 || mine[0].Pin == nil || mine[0].Pin.Point != (geo.Position{36.821947, -1.292066}) || mine[0].Address != "12 Riverside Road" {
		t.Fatalf("raiser's read = %+v (%v)", mine, err)
	}
	seen := mine[0].AsSeenByOthers()
	if seen.Pin.Point != (geo.Position{36.822, -1.292}) || seen.Address != "" || seen.Area != "riverside" {
		t.Fatalf("others' read = %+v", seen)
	}
	if mine[0].Pin.Point != (geo.Position{36.821947, -1.292066}) {
		t.Fatal("projecting changed the raiser's own copy")
	}
}

func TestNearbyListsPublishedItemsOfOneThemeWithinTheDistance(t *testing.T) {
	cm, ctx := newManager(t)
	centre := geo.Position{36.8219, -1.2921}
	for _, r := range []catalog.RaisedItem{
		{Title: "close", Topic: "roads", Publishable: true, Pin: pin(36.8228, -1.2921)},
		{Title: "closest", Topic: "roads", Publishable: true, Pin: pin(36.8220, -1.2921)},
		{Title: "far", Topic: "roads", Publishable: true, Pin: pin(36.9, -1.2921)},
		{Title: "private", Topic: "roads", Pin: pin(36.8220, -1.2921)},
		{Title: "other theme", Topic: "lighting", Publishable: true, Pin: pin(36.8220, -1.2921)},
		{Title: "no pin", Topic: "roads", Publishable: true, Address: "Riverside Road"},
	} {
		r.RaisedBy = "amina"
		raised, err := cm.RaiseItem(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		if raised.Publishable {
			groupOnBoard(t, cm, ctx, raised.ID, false)
		}
	}
	got, err := cm.NearbyRaisedItems(ctx, "roads", centre, 500)
	if err != nil || len(got) != 2 || got[0].Title != "closest" || got[1].Title != "close" {
		t.Fatalf("nearby = %+v (%v)", got, err)
	}
	if got[0].Pin.Point != (geo.Position{36.822, -1.292}) {
		t.Fatalf("a nearby read carried an exact pin: %+v", got[0].Pin)
	}
	for name, call := range map[string]func() error{
		"no theme":    func() error { _, err := cm.NearbyRaisedItems(ctx, "", centre, 500); return err },
		"no distance": func() error { _, err := cm.NearbyRaisedItems(ctx, "roads", centre, 0); return err },
		"negative":    func() error { _, err := cm.NearbyRaisedItems(ctx, "roads", centre, -1); return err },
	} {
		if err := call(); !errors.Is(err, catalog.ErrInvalidRaisedItem) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func groupOnBoard(t *testing.T, cm catalog.CatalogManager, ctx context.Context, raisedItemID string, hidden bool) {
	t.Helper()
	b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "board " + raisedItemID})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.GroupUnderBoardItem(ctx, raisedItemID, b.ID); err != nil {
		t.Fatal(err)
	}
	if hidden {
		if err := cm.SetBoardItemHidden(ctx, b.ID, true); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNearbyListsOnlyItemsOnThePublicBoard(t *testing.T) {
	cm, ctx := newManager(t)
	centre := geo.Position{36.8219, -1.2921}
	place := func(title string) catalog.RaisedItem {
		r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: title, RaisedBy: "amina", Topic: "roads", Publishable: true, Pin: pin(36.8220, -1.2921)})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	place("ungrouped")
	groupOnBoard(t, cm, ctx, place("hidden").ID, true)
	groupOnBoard(t, cm, ctx, place("shown").ID, false)
	ungrouped := place("ungrouped again")
	groupOnBoard(t, cm, ctx, ungrouped.ID, false)
	if err := cm.UngroupFromBoardItem(ctx, ungrouped.ID); err != nil {
		t.Fatal(err)
	}
	got, err := cm.NearbyRaisedItems(ctx, "roads", centre, 500)
	if err != nil || len(got) != 1 || got[0].Title != "shown" {
		t.Fatalf("nearby = %+v (%v), want only the item grouped under a shown board item", got, err)
	}
}

func TestTheStreetComesFromTheMatchedAddressElseThePinElseTheGiven(t *testing.T) {
	cm, ctx := newManager(t)
	cm.UseStreetResolver(func(_ context.Context, at *geo.Position, address string) (string, error) {
		switch {
		case at != nil && at.Lat() > 0:
			return "Pin Street", nil
		case address == "12 Riverside Road":
			return "Riverside Road", nil
		}
		return "", nil
	})
	cases := map[string]struct {
		in   catalog.RaisedItem
		want string
	}{
		"matched address wins over pin":      {catalog.RaisedItem{Pin: pin(36.8, 1.2), Address: "12 Riverside Road", Street: "given"}, "Riverside Road"},
		"unmatched address falls to the pin": {catalog.RaisedItem{Pin: pin(36.8, 1.2), Address: "Nowhere Lane", Street: "given"}, "Pin Street"},
		"nothing matched keeps the given":    {catalog.RaisedItem{Pin: pin(36.8, -1.2), Address: "Nowhere Lane", Street: "given"}, "given"},
		"nothing at all is no street":        {catalog.RaisedItem{}, ""},
	}
	for name, c := range cases {
		c.in.Title, c.in.RaisedBy = name, "amina"
		got, err := cm.RaiseItem(ctx, c.in)
		if err != nil || got.Street != c.want {
			t.Fatalf("%s: street = %q (%v), want %q", name, got.Street, err, c.want)
		}
	}
}

func TestAStreetResolverErrorRefusesTheItem(t *testing.T) {
	cm, ctx := newManager(t)
	cm.UseStreetResolver(func(context.Context, *geo.Position, string) (string, error) { return "", errors.New("down") })
	if _, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Pothole", RaisedBy: "amina", Address: "12 Riverside Road"}); err == nil {
		t.Fatal("a failing street resolver stored the item")
	}
}

func TestOthersReadTheStreetWithTheAreaButNotTheAddress(t *testing.T) {
	cm, ctx := newManager(t)
	cm.UseStreetResolver(func(context.Context, *geo.Position, string) (string, error) { return "Riverside Road", nil })
	if _, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Pothole", RaisedBy: "amina", Pin: pin(36.821947, -1.292066), Address: "12 Riverside Road", Area: "riverside"}); err != nil {
		t.Fatal(err)
	}
	mine, err := cm.ListRaisedItems(ctx, "amina")
	if err != nil || len(mine) != 1 || mine[0].Street != "Riverside Road" || mine[0].Address != "12 Riverside Road" {
		t.Fatalf("raiser's read = %+v (%v)", mine, err)
	}
	seen := mine[0].AsSeenByOthers()
	if seen.Street != "Riverside Road" || seen.Area != "riverside" || seen.Address != "" || seen.Pin.Point != (geo.Position{36.822, -1.292}) {
		t.Fatalf("others' read = %+v", seen)
	}
}
