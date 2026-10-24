package mwanachamacatalog

import (
	"testing"
	"time"
)

func TestTheAnswerClockCountsWeekdaysOnly(t *testing.T) {
	day := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	cases := []struct {
		from string
		n    int
		want string
	}{
		{"2026-10-23T09:00:00Z", 1, "2026-10-22T09:00:00Z"},
		{"2026-10-26T09:00:00Z", 1, "2026-10-23T09:00:00Z"},
		{"2026-10-24T09:00:00Z", 1, "2026-10-23T09:00:00Z"},
		{"2026-10-25T09:00:00Z", 5, "2026-10-19T09:00:00Z"},
		{"2026-10-30T09:00:00Z", 10, "2026-10-16T09:00:00Z"},
	}
	for _, c := range cases {
		if got := weekdaysBefore(day(c.from), c.n); !got.Equal(day(c.want)) {
			t.Errorf("%d weekdays before %s = %s, want %s", c.n, c.from, got.Format(time.RFC3339), c.want)
		}
	}
}
