package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestATimeTakesOnlyAsManyHoldersAsItsCapacity(t *testing.T) {
	cm, ctx := newManager(t)
	s, err := cm.OfferSlot(ctx, catalog.Slot{SubjectID: "svc-1", StartsAt: "2026-11-02T09:00:00Z", EndsAt: "2026-11-02T09:30:00Z", Capacity: 2})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range []string{"amina", "baraka"} {
		if _, err := cm.Book(ctx, s.ID, h); err != nil {
			t.Fatalf("book %s: %v", h, err)
		}
	}
	if _, err := cm.Book(ctx, s.ID, "chege"); !errors.Is(err, catalog.ErrSlotFull) {
		t.Fatalf("third booking: %v, want full", err)
	}
	if _, err := cm.Book(ctx, s.ID, "amina"); !errors.Is(err, catalog.ErrAlreadyBooked) {
		t.Fatalf("second booking by one holder: %v", err)
	}
	if _, err := cm.Book(ctx, "nope", "chege"); !errors.Is(err, catalog.ErrSlotNotFound) {
		t.Fatalf("unknown time: %v", err)
	}
}

func TestACancelledPlaceFreesTheTime(t *testing.T) {
	cm, ctx := newManager(t)
	s, err := cm.OfferSlot(ctx, catalog.Slot{SubjectID: "svc-1", StartsAt: "2026-11-02T09:00:00Z", EndsAt: "2026-11-02T09:30:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := cm.Book(ctx, s.ID, "amina")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.CancelBooking(ctx, b.ID, "baraka", ""); !errors.Is(err, catalog.ErrBookingNotFound) {
		t.Fatalf("another holder cancelling: %v", err)
	}
	got, err := cm.CancelBooking(ctx, b.ID, "amina", "")
	if err != nil || got.CancelledBy != "amina" || got.CancelledAt == "" {
		t.Fatalf("cancel: %+v %v", got, err)
	}
	if _, err := cm.CancelBooking(ctx, b.ID, "amina", ""); !errors.Is(err, catalog.ErrBookingCancelled) {
		t.Fatalf("second cancel: %v", err)
	}
	if _, err := cm.Book(ctx, s.ID, "baraka"); err != nil {
		t.Fatalf("booking a freed time: %v", err)
	}
}

func TestStaffCancelOnlyWithAReason(t *testing.T) {
	cm, ctx := newManager(t)
	s, _ := cm.OfferSlot(ctx, catalog.Slot{SubjectID: "svc-1", StartsAt: "2026-11-02T09:00:00Z", EndsAt: "2026-11-02T09:30:00Z"})
	b, err := cm.Book(ctx, s.ID, "amina")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.CancelBookingAsStaff(ctx, b.ID, "clerk", "  "); !errors.Is(err, catalog.ErrCancelReasonRequired) {
		t.Fatalf("no reason: %v", err)
	}
	got, err := cm.CancelBookingAsStaff(ctx, b.ID, "clerk", "The office is closed")
	if err != nil || got.CancelReason != "The office is closed" || got.CancelledBy != "clerk" {
		t.Fatalf("staff cancel: %+v %v", got, err)
	}
	mine, err := cm.ListHolderBookings(ctx, "amina")
	if err != nil || len(mine) != 1 || mine[0].CancelReason != "The office is closed" {
		t.Fatalf("holder sees %+v %v", mine, err)
	}
}

func TestADaysBookingsAreListedForStaff(t *testing.T) {
	cm, ctx := newManager(t)
	a, _ := cm.OfferSlot(ctx, catalog.Slot{SubjectID: "svc-1", StartsAt: "2026-11-02T10:00:00Z", EndsAt: "2026-11-02T10:30:00Z", Capacity: 3})
	b, _ := cm.OfferSlot(ctx, catalog.Slot{SubjectID: "svc-1", StartsAt: "2026-11-03T09:00:00Z", EndsAt: "2026-11-03T09:30:00Z"})
	c, _ := cm.OfferSlot(ctx, catalog.Slot{SubjectID: "svc-2", StartsAt: "2026-11-02T08:00:00Z", EndsAt: "2026-11-02T08:30:00Z"})
	for _, pair := range [][2]string{{a.ID, "amina"}, {a.ID, "baraka"}, {b.ID, "amina"}, {c.ID, "chege"}} {
		if _, err := cm.Book(ctx, pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}
	day, err := cm.ListDayBookings(ctx, "", "2026-11-02")
	if err != nil || len(day) != 3 || day[0].HolderID != "chege" {
		t.Fatalf("day: %+v %v", day, err)
	}
	one, err := cm.ListDayBookings(ctx, "svc-1", "2026-11-02")
	if err != nil || len(one) != 2 {
		t.Fatalf("one service's day: %+v %v", one, err)
	}
	if _, err := cm.ListDayBookings(ctx, "", "tomorrow"); !errors.Is(err, catalog.ErrInvalidSlot) {
		t.Fatalf("bad day: %v", err)
	}
	slots, err := cm.ListSlots(ctx, "svc-1")
	if err != nil || len(slots) != 2 || slots[0].ID != a.ID {
		t.Fatalf("slots: %+v %v", slots, err)
	}
}

func TestATimeMustEndAfterItStarts(t *testing.T) {
	cm, ctx := newManager(t)
	for _, s := range []catalog.Slot{
		{SubjectID: "svc-1", StartsAt: "2026-11-02T09:30:00Z", EndsAt: "2026-11-02T09:00:00Z"},
		{SubjectID: "", StartsAt: "2026-11-02T09:00:00Z", EndsAt: "2026-11-02T09:30:00Z"},
		{SubjectID: "svc-1", StartsAt: "soon", EndsAt: "2026-11-02T09:30:00Z"},
		{SubjectID: "svc-1", StartsAt: "2026-11-02T09:00:00Z", EndsAt: "2026-11-02T09:30:00Z", Capacity: -1},
	} {
		if _, err := cm.OfferSlot(ctx, s); !errors.Is(err, catalog.ErrInvalidSlot) {
			t.Fatalf("%+v: %v", s, err)
		}
	}
}

func TestADomainThatBooksNoTimesRefusesBooking(t *testing.T) {
	cm, ctx := managerDeclaring(t, "entry", "entry_private", "share_link")
	if _, err := cm.Book(ctx, "x", "amina"); !errors.Is(err, catalog.ErrTimesNotBooked) {
		t.Fatalf("book: %v", err)
	}
}
