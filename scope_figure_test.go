package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestTheCurrentScopeFigureIsTheLatestAsOfNotTheLatestRecorded(t *testing.T) {
	cm, ctx := newManager(t)
	for _, f := range []catalog.ScopeFigure{
		{ScopeID: "s-1", Value: 4200, AsOf: "2024-06-30T00:00:00Z", Source: "survey 2024", RecordedBy: "clerk"},
		{ScopeID: "s-1", Value: 3900, AsOf: "2019-08-24T00:00:00+03:00", Source: "census 2019", RecordedBy: "clerk"},
		{ScopeID: "s-2", Value: 10, AsOf: "2025-01-01T00:00:00Z", Source: "estimate", RecordedBy: "clerk"},
	} {
		if _, err := cm.RecordScopeFigure(ctx, f); err != nil {
			t.Fatal(err)
		}
	}
	cur, err := cm.CurrentScopeFigure(ctx, "s-1")
	if err != nil || cur.Value != 4200 || cur.Source != "survey 2024" {
		t.Fatalf("current = %+v (%v), want the 2024 survey", cur, err)
	}
	all, err := cm.ListScopeFigures(ctx, "s-1")
	if err != nil || len(all) != 2 || all[1].AsOf != "2019-08-23T21:00:00Z" {
		t.Fatalf("history = %+v (%v), want two, newest first, as_of in UTC", all, err)
	}
	if _, err := cm.CurrentScopeFigure(ctx, "s-3"); !errors.Is(err, catalog.ErrScopeFigureNotFound) {
		t.Fatalf("an unknown scope: %v", err)
	}
}

func TestTheStoreMintsAScopeFiguresIdAndRecordedTime(t *testing.T) {
	cm, ctx := newManager(t)
	f, err := cm.RecordScopeFigure(ctx, catalog.ScopeFigure{ID: "forged", CreatedAt: "1999-01-01T00:00:00Z", ScopeID: "s-1", Value: 1, AsOf: "2024-01-01T00:00:00Z", Source: "x", RecordedBy: "clerk"})
	if err != nil || f.ID == "forged" || f.ID == "" || f.CreatedAt == "1999-01-01T00:00:00Z" {
		t.Fatalf("recorded = %+v (%v), want a minted id and time", f, err)
	}
}

func TestAScopeFigureNeedsAScopeADateASourceAndARecorder(t *testing.T) {
	cm, ctx := newManager(t)
	good := catalog.ScopeFigure{ScopeID: "s-1", Value: 1, AsOf: "2024-01-01T00:00:00Z", Source: "x", RecordedBy: "clerk"}
	for name, mutate := range map[string]func(*catalog.ScopeFigure){
		"no scope":       func(f *catalog.ScopeFigure) { f.ScopeID = "" },
		"no date":        func(f *catalog.ScopeFigure) { f.AsOf = "" },
		"not a date":     func(f *catalog.ScopeFigure) { f.AsOf = "last year" },
		"no source":      func(f *catalog.ScopeFigure) { f.Source = "" },
		"no recorder":    func(f *catalog.ScopeFigure) { f.RecordedBy = "" },
		"negative value": func(f *catalog.ScopeFigure) { f.Value = -1 },
	} {
		f := good
		mutate(&f)
		if _, err := cm.RecordScopeFigure(ctx, f); !errors.Is(err, catalog.ErrInvalidScopeFigure) {
			t.Errorf("%s: %v, want ErrInvalidScopeFigure", name, err)
		}
	}
}
