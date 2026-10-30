package routes_test

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/routes"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func TestCAT21_FulfilDoesNotUndoAReviewersReject(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:cat21?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(2)
	t.Cleanup(func() { sqlDB.Close() })

	s, err := catalog.LoadSpec(filepath.Join("..", "spec", "examples", "agency.catalog.json"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cm, err := catalog.NewCatalogManager(db, s)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	h := http.NewServeMux()
	for _, r := range routes.Routes(cm) {
		h.Handle(r.Method+" "+r.Path, r.Handler)
	}

	if rec := do(t, h, "PUT", "/entries/answer", `{"name":"Answer","doc":"{\"a\":1}"}`); rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("precondition: PUT /entries/answer answered %d %s", rec.Code, rec.Body.String())
	}
	filed := do(t, h, "POST", "/suggestions", `{"slug":"clinic","title":"Build clinic","why":"worth having","source":"run"}`)
	var sg struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(filed.Body.Bytes(), &sg); err != nil || filed.Code != http.StatusCreated || sg.ID == "" {
		t.Fatalf("precondition: POST /suggestions answered %d %s", filed.Code, filed.Body.String())
	}
	if rec := do(t, h, "POST", "/suggestions/"+sg.ID+"/decide", `{"status":"approved"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: approving answered %d %s", rec.Code, rec.Body.String())
	}

	suggestions, ok := s.ByRole("suggestion")
	if !ok {
		t.Fatal("the shipped spec fills no suggestion role")
	}
	suggestionTable := s.TableFor(suggestions)

	var rejectAnswer string
	fired := false
	err = db.Callback().Update().Before("gorm:update").Register("cat21_reject_mid_fulfil", func(tx *gorm.DB) {
		row, isRow := tx.Statement.Dest.(map[string]any)
		if fired || tx.Statement.Table != suggestionTable || !isRow {
			return
		}
		if row["status"] != "built" {
			return
		}
		fired = true
		rec := do(t, h, "POST", "/suggestions/"+sg.ID+"/decide", `{"status":"rejected"}`)
		if rec.Code == http.StatusOK {
			rejectAnswer = rec.Body.String()
		}
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}

	fulfil := do(t, h, "POST", "/suggestions/"+sg.ID+"/fulfil", `{"slug":"answer","run_ref":"run-1"}`)
	if !fired {
		t.Fatalf("precondition: POST /fulfil never reached its status write (answered %d %s)", fulfil.Code, fulfil.Body.String())
	}
	var reviewer struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(rejectAnswer), &reviewer); err != nil || reviewer.Status != "rejected" {
		t.Fatalf("precondition: the reviewer's POST /decide {\"status\":\"rejected\"} did not answer 200 with status rejected (got %q)", rejectAnswer)
	}

	after := do(t, h, "GET", "/suggestions/"+sg.ID, "")
	var got struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(after.Body.Bytes(), &got); err != nil {
		t.Fatalf("GET /suggestions/%s answered %d %s", sg.ID, after.Code, after.Body.String())
	}
	if got.Status != "rejected" {
		t.Errorf("CAT21 (P2): a reviewer's POST /decide {\"status\":\"rejected\"} answered 200 with status rejected while a POST /fulfil was in flight, the fulfil answered %d, and the suggestion now reads %q; want it to stay rejected, because a suggestion may only be marked built while it is still approved at the moment that write lands", fulfil.Code, got.Status)
	}
}
