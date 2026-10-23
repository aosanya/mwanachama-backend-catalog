package mwanachamacatalog_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func TestASecondSendReplacesTheHoldersRecordAndKeepsEveryChangeTime(t *testing.T) {
	cm, ctx := newManager(t)
	first, err := cm.PutRecord(ctx, catalog.Record{SubjectID: "form-1", HolderID: "amina", Doc: `{"agree": true}`})
	if err != nil {
		t.Fatal(err)
	}
	second, err := cm.PutRecord(ctx, catalog.Record{ID: "forged", SubjectID: "form-1", HolderID: "amina", Doc: `{"agree": false, "amount": 40}`})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || second.CreatedAt != first.CreatedAt {
		t.Fatalf("second send minted a new record: %+v then %+v", first, second)
	}
	records, err := cm.ListRecords(ctx, "form-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Doc == `{"agree": true}` {
		t.Fatalf("records = %+v", records)
	}
	var changes []string
	if err := json.Unmarshal([]byte(records[0].Changes), &changes); err != nil || len(changes) != 2 {
		t.Fatalf("changes = %q (%v)", records[0].Changes, err)
	}
}

func TestCountsAreWorkedOutFromEachSubjectsLatestRecordsOnly(t *testing.T) {
	cm, ctx := newManager(t)
	for _, r := range []catalog.Record{
		{SubjectID: "form-1", HolderID: "amina", Doc: `{"agree": true, "amount": 10}`},
		{SubjectID: "form-1", HolderID: "amina", Doc: `{"agree": false, "amount": 30}`},
		{SubjectID: "form-1", HolderID: "baraka", Doc: `{"agree": false, "amount": 50}`},
		{SubjectID: "form-2", HolderID: "baraka", Doc: `{"agree": true}`},
	} {
		if _, err := cm.PutRecord(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	tallies, err := cm.CountRecords(ctx, "form-1", "")
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]catalog.FieldTally{}
	for _, tally := range tallies {
		by[tally.Field] = tally
	}
	if a := by["agree"]; a.Yes != 0 || a.No != 2 {
		t.Fatalf("agree = %+v", a)
	}
	if n := by["amount"]; n.Numbers != 2 || *n.Average != 40 || *n.Lowest != 30 || *n.Highest != 50 {
		t.Fatalf("amount = %+v", n)
	}
}

func TestARecordNeedsASubjectAHolderAndAnObject(t *testing.T) {
	cm, ctx := newManager(t)
	for _, r := range []catalog.Record{
		{HolderID: "amina", Doc: `{}`},
		{SubjectID: "form-1", Doc: `{}`},
		{SubjectID: "form-1", HolderID: "amina", Doc: `[1]`},
	} {
		if _, err := cm.PutRecord(ctx, r); !errors.Is(err, catalog.ErrInvalidRecord) {
			t.Fatalf("PutRecord(%+v) = %v, want ErrInvalidRecord", r, err)
		}
	}
}

func TestADomainThatKeepsNoRecordsStillBuildsAndRefusesThem(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(".", "spec", "examples", "agency.catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var kept []any
	for _, o := range doc["objects"].([]any) {
		if o.(map[string]any)["role"] != "record" {
			kept = append(kept, o)
		}
	}
	doc["objects"] = kept
	trimmed, _ := json.Marshal(doc)
	s, err := catalog.ParseSpec(trimmed)
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatal(err)
	}
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("a spec without records must still build: %v", err)
	}
	if _, err := cm.ListRecords(t.Context(), "form-1", ""); !errors.Is(err, catalog.ErrRecordsNotKept) {
		t.Fatalf("ListRecords = %v, want ErrRecordsNotKept", err)
	}
}

func TestAHolderSendingUnderTwoStandingsKeepsTwoRecordsCountedUnderEach(t *testing.T) {
	cm, ctx := newManager(t)
	for _, r := range []catalog.Record{
		{SubjectID: "form-1", HolderID: "amina", ScopeID: "traders", Doc: `{"agree": true}`},
		{SubjectID: "form-1", HolderID: "amina", ScopeID: "teachers", Doc: `{"agree": false}`},
		{SubjectID: "form-1", HolderID: "amina", ScopeID: "traders", Doc: `{"agree": false}`},
		{SubjectID: "form-1", HolderID: "baraka", ScopeID: "traders", Doc: `{"agree": true}`},
	} {
		if _, err := cm.PutRecord(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	all, err := cm.ListRecords(ctx, "form-1", "")
	if err != nil || len(all) != 3 {
		t.Fatalf("records = %+v (%v), want one per holder per standing", all, err)
	}
	agree := func(scope string) catalog.FieldTally {
		tallies, err := cm.CountRecords(ctx, "form-1", scope)
		if err != nil || len(tallies) != 1 {
			t.Fatalf("CountRecords(%q) = %+v (%v)", scope, tallies, err)
		}
		return tallies[0]
	}
	if got := agree("traders"); got.Yes != 1 || got.No != 1 || got.Counted != 2 || got.Holders != 2 {
		t.Errorf("traders = %+v, want amina's resend and baraka", got)
	}
	if got := agree("teachers"); got.Yes != 0 || got.No != 1 || got.Counted != 1 || got.Holders != 1 {
		t.Errorf("teachers = %+v, want amina once", got)
	}
	if got := agree(""); got.Yes != 1 || got.No != 2 || got.Counted != 3 || got.Holders != 2 {
		t.Errorf("overall = %+v, want amina counted under each standing", got)
	}
}

func TestAnOverallCountCountsAPersonAnsweringAsTwoGroupsOnce(t *testing.T) {
	cm, ctx := newManager(t)
	for _, scope := range []string{"traders", "teachers"} {
		if _, err := cm.PutRecord(ctx, catalog.Record{SubjectID: "form-1", HolderID: "amina", ScopeID: scope, Doc: `{"agree": true}`}); err != nil {
			t.Fatal(err)
		}
	}
	for _, scope := range []string{"", "traders", "teachers"} {
		tallies, err := cm.CountRecords(ctx, "form-1", scope)
		if err != nil || len(tallies) != 1 {
			t.Fatalf("CountRecords(%q) = %+v (%v)", scope, tallies, err)
		}
		if tallies[0].Holders != 1 {
			t.Errorf("CountRecords(%q).Holders = %d, want amina once", scope, tallies[0].Holders)
		}
	}
}
