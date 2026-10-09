package mwanachamacatalog_test

import (
	"fmt"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT44_OpenBug_ListShareLinksHasNoDefaultCap(t *testing.T) {
	cm, ctx := newManager(t)
	seed(t, cm, ctx, "probe-entry", catalog.VisibilityPrivate)

	const issued = 150
	for i := 0; i < issued; i++ {
		if _, _, err := cm.IssueShareLink(ctx, "probe-entry", fmt.Sprintf("label-%d", i), "admin"); err != nil {
			t.Fatalf("issue %d: %v", i, err)
		}
	}

	got, err := cm.ListShareLinks(ctx, "probe-entry")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) >= issued {
		t.Fatalf("CAT44 open bug: ListShareLinks(\"probe-entry\") with no limit returned all %d of %d issued keys; a caller who names no limit should get a capped default page, not every key ever issued for the entry", len(got), issued)
	}
}
