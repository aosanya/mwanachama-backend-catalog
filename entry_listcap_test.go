package mwanachamacatalog_test

import (
	"fmt"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT10_ListEntriesAppliesADefaultCap(t *testing.T) {
	cm, ctx := newManager(t)

	const seeded = 150
	for i := 0; i < seeded; i++ {
		if _, err := cm.CreateEntry(ctx, catalog.Entry{
			Slug:    fmt.Sprintf("agency-%03d", i),
			Name:    fmt.Sprintf("Agency %03d", i),
			Summary: "a summary",
			Doc:     doc,
		}); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	got, err := cm.ListEntries(ctx, catalog.ListFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) >= seeded {
		t.Fatalf("CAT10 open bug: ListEntries(ListFilter{}) with no limit returned all %d seeded entries; a caller who names no limit should get a capped default page, not the whole live table", len(got))
	}
}
