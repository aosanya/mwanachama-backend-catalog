package mwanachamacatalog_test

import (
	"errors"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestARaisedItemNamingAnEntryIsListedUnderThatSubject(t *testing.T) {
	cm, ctx := newManager(t)
	holder, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "holder-one", Name: "Holder one", Doc: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	known, err := cm.FileRaisedItem(ctx, catalog.RaisedItem{Title: "Broken streetlight", RaisedBy: "amina", FiledBy: "clerk-2", Channel: "phone", SubjectID: holder.ID})
	if err != nil {
		t.Fatal(err)
	}
	filed, err := cm.FileRaisedItem(ctx, catalog.RaisedItem{Title: "Market drainage", RaisedBy: "baraka", FiledBy: "clerk-1", Channel: "counter", SubjectID: holder.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Pothole", RaisedBy: "contact:0700"}); err != nil {
		t.Fatal(err)
	}
	listed, err := cm.ListRaisedItemsOfSubject(ctx, holder.ID)
	ids := map[string]bool{}
	for _, r := range listed {
		if r.SubjectID == holder.ID {
			ids[r.ID] = true
		}
	}
	if err != nil || len(listed) != 2 || !ids[known.ID] || !ids[filed.ID] {
		t.Fatalf("subject's items = %+v (%v)", listed, err)
	}
	mine, err := cm.ListRaisedItems(ctx, "amina")
	if err != nil || len(mine) != 1 || mine[0].SubjectID != holder.ID {
		t.Fatalf("raiser's read lost the subject: %+v (%v)", mine, err)
	}
}

func TestARaisedItemWithNoSubjectIsTiedToNobody(t *testing.T) {
	cm, ctx := newManager(t)
	r, err := cm.RaiseItem(ctx, catalog.RaisedItem{Title: "Dark road", RaisedBy: "contact:0700"})
	if err != nil || r.SubjectID != "" {
		t.Fatalf("unknown raiser: %+v (%v)", r, err)
	}
	if _, err := cm.ListRaisedItemsOfSubject(ctx, ""); !errors.Is(err, catalog.ErrInvalidRaisedItem) {
		t.Fatalf("empty subject listed: %v", err)
	}
}

func TestARaisedItemNamingNoEntryInThisCatalogIsRefused(t *testing.T) {
	cm, ctx := newManager(t)
	gone, err := cm.CreateEntry(ctx, catalog.Entry{Slug: "gone", Name: "Gone", Doc: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	if err := cm.DeleteEntry(ctx, "gone"); err != nil {
		t.Fatal(err)
	}
	for name, subject := range map[string]string{"unknown": "no-such-entry", "deleted": gone.ID} {
		if _, err := cm.FileRaisedItem(ctx, catalog.RaisedItem{Title: "Potholes", RaisedBy: "amina", FiledBy: "clerk-1", Channel: "phone", SubjectID: subject}); !errors.Is(err, catalog.ErrEntryNotFound) {
			t.Errorf("%s file: %v", name, err)
		}
	}
	if mine, err := cm.ListRaisedItems(ctx, "amina"); err != nil || len(mine) != 0 {
		t.Fatalf("a refused raise was stored: %+v (%v)", mine, err)
	}
}
