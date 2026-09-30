package routes_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAToolRefusesAServerOwnedFieldInAnyCasingAndStoresNothing(t *testing.T) {
	tools, cm, ctx := newTools(t)

	forged := []struct {
		tool, args, key string
	}{
		{"catalog_suggestion_file", `{"slug":"wanted","title":"W","source":"owner","proposal":"{}","Status":"built"}`, "Status"},
		{"catalog_suggestion_file", `{"slug":"wanted","title":"W","source":"owner","proposal":"{}","ATTEMPTS":42}`, "ATTEMPTS"},
		{"catalog_suggestion_file", `{"slug":"wanted","title":"W","source":"owner","proposal":"{}","Created_At":"1999-01-01T00:00:00Z"}`, "Created_At"},
		{"catalog_entry_upsert", `{"slug":"alpha","name":"A","doc":"{}","Visibility":"public"}`, "Visibility"},
		{"catalog_entry_upsert", `{"slug":"alpha","name":"A","doc":"{}","Approved_At":"1999-01-01T00:00:00Z"}`, "Approved_At"},
		{"catalog_entry_upsert", `{"slug":"alpha","name":"A","doc":"{}","SubmittedBy":"mallory"}`, "SubmittedBy"},
		{"catalog_entry_upsert", `{"slug":"alpha","Slug":"beta","name":"A","doc":"{}"}`, "Slug"},
		{"catalog_run_record", `{"run_id":"r1","summary":"s","templates_filed":1,"Submitted_By":"mallory"}`, "Submitted_By"},
	}
	for _, f := range forged {
		_, err := tools[f.tool].Invoke(ctx, json.RawMessage(f.args))
		if err == nil || !strings.Contains(err.Error(), `takes no argument "`+f.key+`"`) {
			t.Errorf("%s with %s answered err=%v, want the call refused naming %q", f.tool, f.args, err, f.key)
		}
	}

	if got, err := cm.ListSuggestions(ctx); err != nil || len(got) != 0 {
		t.Errorf("a refused suggestion_file stored %d suggestions (err %v), want 0", len(got), err)
	}
	if _, err := cm.GetEntry(ctx, "alpha"); err == nil {
		t.Errorf("a refused entry_upsert stored entry alpha")
	}
	if _, err := cm.GetEntry(ctx, "beta"); err == nil {
		t.Errorf("a refused entry_upsert stored entry beta")
	}
	if got, err := cm.ListRuns(ctx, 0); err != nil || len(got) != 0 {
		t.Errorf("a refused run_record stored %d runs (err %v), want 0", len(got), err)
	}

	invoke(t, tools["catalog_suggestion_file"], ctx, `{"slug":"wanted","title":"W","source":"owner","proposal":"{}"}`)
	invoke(t, tools["catalog_entry_upsert"], ctx, `{"slug":"alpha","name":"A","doc":"{}"}`)
	sugs, err := cm.ListSuggestions(ctx)
	if err != nil || len(sugs) != 1 || sugs[0].Status != "proposed" || sugs[0].Attempts != 0 {
		t.Errorf("the same suggestion without forged fields stored %+v (err %v), want one proposed suggestion with 0 attempts", sugs, err)
	}
	entry, err := cm.GetEntry(ctx, "alpha")
	if err != nil || entry.Visibility != "pending" || entry.ApprovedAt != "" {
		t.Errorf("the same entry without forged fields read %+v (err %v), want pending and unapproved", entry, err)
	}
}
