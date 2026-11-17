package models

import "testing"

func TestNormalizePublishAtAcceptsEmptyAndConvertsToUTC(t *testing.T) {
	got, ok := NormalizePublishAt("")
	if !ok || got != "" {
		t.Fatalf("empty = %q, %v", got, ok)
	}
	got, ok = NormalizePublishAt("2026-03-01T10:00:00+03:00")
	if !ok || got != "2026-03-01T07:00:00Z" {
		t.Fatalf("offset = %q, %v", got, ok)
	}
}

func TestNormalizePublishAtRefusesNonRFC3339(t *testing.T) {
	for _, bad := range []string{"2026-03-01", "tomorrow", "2026-03-01 10:00:00", "2026-13-01T00:00:00Z"} {
		if got, ok := NormalizePublishAt(bad); ok || got != "" {
			t.Fatalf("%q = %q, %v", bad, got, ok)
		}
	}
}

func TestEntryPublishedByIsOpenWhenUnscheduledAndInclusiveAtTheInstant(t *testing.T) {
	now := "2026-03-01T07:00:00Z"
	cases := map[string]bool{
		"":                     true,
		"2026-03-01T06:59:59Z": true,
		"2026-03-01T07:00:00Z": true,
		"2026-03-01T07:00:01Z": false,
	}
	for publishAt, want := range cases {
		if got := (Entry{PublishAt: publishAt}).PublishedBy(now); got != want {
			t.Fatalf("publish_at %q = %v, want %v", publishAt, got, want)
		}
	}
}

func TestEntryRetainedAtEndsExactlyAtTheDisposeInstant(t *testing.T) {
	now := "2026-03-01T07:00:00Z"
	cases := map[string]bool{
		"":                     true,
		"2026-03-01T07:00:01Z": true,
		"2026-03-01T07:00:00Z": false,
		"2026-02-28T00:00:00Z": false,
	}
	for disposeAt, want := range cases {
		if got := (Entry{DisposeAt: disposeAt}).RetainedAt(now); got != want {
			t.Fatalf("dispose_at %q = %v, want %v", disposeAt, got, want)
		}
	}
}

func TestDocFilterValidRefusesPathsThatCouldReachSQL(t *testing.T) {
	good := []string{"sector", "_template.sector", "identifiers.isbn", "a-b.c_d.9"}
	bad := []string{"", ".a", "a.", "a..b", "a b", "a'b", "a;drop", "a->b", "a.b)", "a/b"}
	for _, p := range good {
		if !(DocFilter{Path: p}).Valid() {
			t.Fatalf("%q should be valid", p)
		}
	}
	for _, p := range bad {
		if (DocFilter{Path: p}).Valid() {
			t.Fatalf("%q should be refused", p)
		}
	}
}

func TestDocFilterSegmentsSplitsOnDots(t *testing.T) {
	got := (DocFilter{Path: "_template.sector.name"}).Segments()
	if len(got) != 3 || got[0] != "_template" || got[1] != "sector" || got[2] != "name" {
		t.Fatalf("segments = %v", got)
	}
	if one := (DocFilter{Path: "sector"}).Segments(); len(one) != 1 || one[0] != "sector" {
		t.Fatalf("single = %v", one)
	}
}

func TestPageViewSlugFitsPageOnlyWhenIndexHasNoneAndEntryHasOne(t *testing.T) {
	cases := []struct {
		view PageView
		want bool
	}{
		{PageView{Page: PageKindIndex}, true},
		{PageView{Page: PageKindIndex, Slug: "x"}, false},
		{PageView{Page: PageKindEntry, Slug: "x"}, true},
		{PageView{Page: PageKindEntry}, false},
	}
	for _, c := range cases {
		if got := c.view.SlugFitsPage(); got != c.want {
			t.Fatalf("%+v = %v, want %v", c.view, got, c.want)
		}
	}
}

func TestHashKeyIsTheSHA256HexAndNeverTheRawKey(t *testing.T) {
	const emptyDigest = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got := HashKey(""); got != emptyDigest {
		t.Fatalf("empty = %s", got)
	}
	a, b := HashKey("key-one"), HashKey("key-two")
	if a == b || len(a) != 64 || a != HashKey("key-one") {
		t.Fatalf("hashes = %s %s", a, b)
	}
}

func TestIsAssigneeKindAcceptsOnlyActorAndRole(t *testing.T) {
	for kind, want := range map[string]bool{"actor": true, "role": true, "": false, "Actor": false, "group": false} {
		if got := IsAssigneeKind(kind); got != want {
			t.Fatalf("%q = %v, want %v", kind, got, want)
		}
	}
}

func TestRaisedItemFileIsApprovedForPublishingOnlyOnceApprovedAtIsSet(t *testing.T) {
	if (RaisedItemFile{ApprovedBy: "someone"}).ApprovedForPublishing() {
		t.Fatal("no approved_at must not count as approved")
	}
	if !(RaisedItemFile{ApprovedAt: "2026-03-01T07:00:00Z"}).ApprovedForPublishing() {
		t.Fatal("approved_at set must count as approved")
	}
}
