package mwanachamacatalog_test

import (
	"context"
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

const controlByte = "a\x01b"

func TestStaffNotesWithControlBytesAreRefusedWithTheirInvalidError(t *testing.T) {
	ctx := context.Background()
	cases := map[string]struct {
		call func(t *testing.T) error
		want error
	}{
		"run summary": {func(t *testing.T) error {
			_, err := entryManager(t).RecordRun(ctx, catalog.MiningRun{Summary: controlByte})
			return err
		}, catalog.ErrInvalidRun},
		"run reference": {func(t *testing.T) error {
			_, err := entryManager(t).RecordRun(ctx, catalog.MiningRun{RunRef: controlByte})
			return err
		}, catalog.ErrInvalidRun},
		"setting key": {func(t *testing.T) error {
			_, err := entryManager(t).PutSetting(ctx, controlByte, "v")
			return err
		}, catalog.ErrInvalidSetting},
		"setting value": {func(t *testing.T) error {
			_, err := entryManager(t).PutSetting(ctx, "k", controlByte)
			return err
		}, catalog.ErrInvalidSetting},
		"share link label": {func(t *testing.T) error {
			cm, ctx := newManager(t)
			seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)
			_, _, err := cm.IssueShareLink(ctx, "private-one", controlByte, "owner")
			return err
		}, catalog.ErrInvalidShareLink},
		"cancel reason": {func(t *testing.T) error {
			cm := bookingManager(t)
			slot := offered(t, cm, "clinic", "2026-03-02T08:00:00Z", "2026-03-02T09:00:00Z", 1)
			booked, err := cm.Book(ctx, slot.ID, "resident-1")
			if err != nil {
				t.Fatal(err)
			}
			_, err = cm.CancelBookingAsStaff(ctx, booked.ID, "staff-1", controlByte)
			return err
		}, catalog.ErrInvalidSlot},
		"board outcome": {func(t *testing.T) error {
			cm := entryManager(t)
			b, err := cm.PutOnBoard(ctx, catalog.BoardItem{Title: "Potholes"})
			if err != nil {
				t.Fatal(err)
			}
			return cm.SetBoardItemOutcome(ctx, b.ID, controlByte)
		}, catalog.ErrInvalidBoardItem},
		"department": {func(t *testing.T) error {
			cm := entryManager(t)
			return cm.ReassignRaisedItem(ctx, raise(t, cm, "Pothole", false).ID, controlByte)
		}, catalog.ErrInvalidRaisedItem},
		"attempt comment": {func(t *testing.T) error {
			cm, ctx := newManager(t)
			s := approve(t, cm, ctx, file(t, cm, ctx, "the-answer").ID)
			_, err := cm.RecordSuggestionAttempt(ctx, s.ID, "run-1", controlByte)
			return err
		}, catalog.ErrInvalidComment},
		"attempt run reference": {func(t *testing.T) error {
			cm, ctx := newManager(t)
			s := approve(t, cm, ctx, file(t, cm, ctx, "the-answer").ID)
			_, err := cm.RecordSuggestionAttempt(ctx, s.ID, controlByte, "tried")
			return err
		}, catalog.ErrInvalidComment},
		"fulfil note": {func(t *testing.T) error {
			cm, ctx := newManager(t)
			s := approve(t, cm, ctx, file(t, cm, ctx, "the-answer").ID)
			seed(t, cm, ctx, "the-answer", catalog.VisibilityPublic)
			_, err := cm.FulfilSuggestion(ctx, s.ID, "the-answer", "run-3", controlByte)
			return err
		}, catalog.ErrInvalidComment},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if err := c.call(t); !errors.Is(err, c.want) {
				t.Fatalf("got %v, want %v", err, c.want)
			}
		})
	}
}
