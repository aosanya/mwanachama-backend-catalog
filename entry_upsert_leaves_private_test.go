package mwanachamacatalog_test

import (
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT39_OpenBug_UpsertLeavingPrivateNeverRetiresItsKeys(t *testing.T) {
	cm, ctx := newManager(t)
	e := seed(t, cm, ctx, "zz-slug", catalog.VisibilityPrivate)

	_, raw, err := cm.IssueShareLink(ctx, "zz-slug", "handed out", "admin")
	if err != nil {
		t.Fatalf("IssueShareLink: %v", err)
	}
	if _, err := cm.OpenEntry(ctx, "zz-slug", raw); err != nil {
		t.Fatalf("precondition: key should open the private entry, got %v", err)
	}

	changed := e
	changed.Doc = `{"agency":{"name":"Changed"},"_template":{"sector":"health"},"modules":[{"code":"auth","enabled":true}]}`
	changed.Summary = "a changed summary"
	got, err := cm.UpsertEntry(ctx, changed)
	if err != nil {
		t.Fatalf("UpsertEntry with a changed document: %v", err)
	}
	if got.Visibility != catalog.VisibilityPending {
		t.Fatalf("precondition: a changed document should force pending, got %q", got.Visibility)
	}

	if _, err := cm.SetVisibility(ctx, "zz-slug", catalog.VisibilityPrivate); err != nil {
		t.Fatalf("precondition: back to private: %v", err)
	}

	if _, err := cm.OpenEntry(ctx, "zz-slug", raw); err == nil {
		t.Fatalf(
			"CAT39 open bug: a share key issued while %q was private still opens it after "+
				"UpsertEntry's changed-document path silently forced it to pending (leaving "+
				"private, the same transition SetVisibility and DeleteEntry both retire every "+
				"live key for) and the entry was then set private again; OpenEntry answered "+
				"err=nil, want ErrEntryNotFound, because leaving private must retire every key "+
				"regardless of which operation causes the entry to leave it",
			"zz-slug")
	}
}
