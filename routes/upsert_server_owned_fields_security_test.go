package routes_test

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestUpsertOverRESTNeverTakesTheIdClockSubmitterOrDeletedFromTheBody(t *testing.T) {
	h, cm, _ := newMountedServer(t)
	q := strconv.Quote(doc)

	forgedCreate := `{"name":"A","id":"forged-id","created_at":"2000-01-01T00:00:00Z","updated_at":"2000-01-01T00:00:00Z","submitted_by":"mallory","deleted":true,"doc":` + q + `}`
	if rec := do(t, h, http.MethodPut, "/entries/owned", forgedCreate); rec.Code != http.StatusOK {
		t.Fatalf("precondition: an operator upsert answered %d %s", rec.Code, rec.Body.String())
	}
	created, err := cm.GetEntry(context.Background(), "owned")
	if err != nil {
		t.Fatalf("an upsert whose body claimed deleted=true left no readable entry: %v", err)
	}
	if created.ID == "forged-id" || created.CreatedAt == "2000-01-01T00:00:00Z" || created.UpdatedAt == "2000-01-01T00:00:00Z" || created.SubmittedBy == "mallory" || created.Deleted {
		t.Errorf("PUT /entries/owned stored body-set server fields: id=%q created_at=%q updated_at=%q submitted_by=%q deleted=%v; want a minted id, the server clock, the signed-in caller and deleted=false", created.ID, created.CreatedAt, created.UpdatedAt, created.SubmittedBy, created.Deleted)
	}

	if rec := do(t, h, http.MethodPost, "/entries/owned/visibility", `{"visibility":"public"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: publishing the entry answered %d %s", rec.Code, rec.Body.String())
	}
	forgedResend := `{"name":"A","id":"other-id","created_at":"1999-01-01T00:00:00Z","submitted_by":"mallory","deleted":true,"doc":` + q + `}`
	if rec := do(t, h, http.MethodPut, "/entries/owned", forgedResend); rec.Code != http.StatusOK {
		t.Fatalf("precondition: re-sending the unchanged document answered %d %s", rec.Code, rec.Body.String())
	}
	after, err := cm.GetEntry(context.Background(), "owned")
	if err != nil {
		t.Fatalf("a re-send whose body claimed deleted=true removed the entry: %v", err)
	}
	if after.ID != created.ID || after.CreatedAt != created.CreatedAt || after.SubmittedBy != created.SubmittedBy || after.Deleted {
		t.Errorf("a re-send rewrote server fields: id %q→%q created_at %q→%q submitted_by %q→%q deleted=%v; want them unchanged", created.ID, after.ID, created.CreatedAt, after.CreatedAt, created.SubmittedBy, after.SubmittedBy, after.Deleted)
	}
	if rec := do(t, h, http.MethodGet, "/public/entries", ""); !strings.Contains(rec.Body.String(), `"slug":"owned"`) {
		t.Errorf("a body claiming deleted=true took a published entry off the anonymous listing: %s", rec.Body.String())
	}
}
