package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestUpsertNeverPublishesAnEntryFromTheRequestBody(t *testing.T) {
	h, cm, tools := newMountedServer(t)

	for _, body := range []string{
		`{"name":"Forged","visibility":"public","doc":` + strconv.Quote(doc) + `}`,
		`{"name":"Forged","Visibility":"public","doc":` + strconv.Quote(doc) + `}`,
		`{"name":"Forged","approved_at":"2000-01-01T00:00:00Z","visibility":"public","doc":` + strconv.Quote(doc) + `}`,
	} {
		if rec := do(t, h, http.MethodPut, "/entries/forged", body); rec.Code != http.StatusOK {
			t.Fatalf("precondition: an operator upsert answered %d %s", rec.Code, rec.Body.String())
		}
		stored, err := cm.GetEntry(context.Background(), "forged")
		if err != nil {
			t.Fatalf("precondition: the upserted entry could not be read back: %v", err)
		}
		if stored.Visibility != "pending" || stored.ApprovedAt != "" {
			t.Errorf("an upsert with body %s stored visibility=%q approved_at=%q; want pending and unapproved until set_visibility", body, stored.Visibility, stored.ApprovedAt)
		}
		if rec := do(t, h, http.MethodGet, "/entries/forged", ""); rec.Code != http.StatusNotFound {
			t.Errorf("an anonymous open of an entry published only by its own upsert body answered %d %s; want 404", rec.Code, rec.Body.String())
		}
		if rec := do(t, h, http.MethodGet, "/public/entries", ""); strings.Contains(rec.Body.String(), "forged") {
			t.Errorf("the anonymous listing carried an entry published only by its own upsert body: %s", rec.Body.String())
		}
	}

	if rec := do(t, h, http.MethodPost, "/entries/forged/visibility", `{"visibility":"public"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: publishing the entry answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, h, http.MethodGet, "/public/entries", ""); !strings.Contains(rec.Body.String(), "forged") {
		t.Fatalf("precondition: the published entry is missing from the anonymous listing: %s", rec.Body.String())
	}
	if rec := do(t, h, http.MethodPut, "/entries/forged", `{"name":"Forged","visibility":"public","doc":"{\"changed\":true}"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: re-sending a changed document answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, h, http.MethodGet, "/public/entries", ""); strings.Contains(rec.Body.String(), "forged") {
		t.Errorf("a changed document claiming visibility public stayed on the anonymous listing: %s", rec.Body.String())
	}
	if rec := do(t, h, http.MethodGet, "/entries/forged", ""); rec.Code != http.StatusNotFound {
		t.Errorf("a changed document claiming visibility public still opened anonymously: %d %s", rec.Code, rec.Body.String())
	}

	upsert := toolNamed(t, tools, "catalog_entry_upsert")
	for _, field := range []string{`"visibility":"public"`, `"approved_at":"2000-01-01T00:00:00Z"`} {
		_, err := upsert(context.Background(), json.RawMessage(`{"slug":"forged-over-mcp","name":"Forged",`+field+`,"doc":`+strconv.Quote(doc)+`}`))
		if err == nil {
			t.Errorf("catalog_entry_upsert accepted %s; want a refusal", field)
		}
	}
	if _, err := cm.GetEntry(context.Background(), "forged-over-mcp"); err == nil {
		t.Errorf("a refused catalog_entry_upsert still wrote the entry")
	}
}
