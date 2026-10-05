package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestDeletedEntry_EveryDoorAnswersMissingAndAResendRevivesNoReviewOrKey(t *testing.T) {
	operator, anon, _ := municipalityWard(t)
	body := `{"name":"Blocked drain","summary":"Reported","doc":"{\"_item\":{\"ward\":\"Kibera North\"}}"}`

	if rec := do(t, operator, http.MethodPut, "/entries/blocked-drain", body); rec.Code != http.StatusOK {
		t.Fatalf("precondition: PUT /entries/blocked-drain answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, operator, http.MethodPost, "/entries/blocked-drain/visibility", `{"visibility":"private"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: making blocked-drain private answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, operator, http.MethodPut, "/entries/blocked-drain/private", `{"readme":"resident-phone-marker"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: PUT /entries/blocked-drain/private answered %d %s", rec.Code, rec.Body.String())
	}
	rec := do(t, operator, http.MethodPost, "/entries/blocked-drain/links", `{"label":"ward office"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("precondition: issuing a share link answered %d %s", rec.Code, rec.Body.String())
	}
	var issued struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &issued); err != nil || issued.Key == "" {
		t.Fatalf("precondition: the share link answer carried no key: %v %s", err, rec.Body.String())
	}
	if rec := do(t, anon, http.MethodGet, "/entries/blocked-drain?k="+issued.Key, ""); rec.Code != http.StatusOK {
		t.Fatalf("precondition: the keyed anonymous open answered %d before the delete", rec.Code)
	}
	if rec := do(t, operator, http.MethodPost, "/entries/blocked-drain/visibility", `{"visibility":"public"}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: approving blocked-drain answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, operator, http.MethodDelete, "/entries/blocked-drain", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /entries/blocked-drain answered %d %s", rec.Code, rec.Body.String())
	}

	operatorDoors := [][3]string{
		{http.MethodGet, "/entries/blocked-drain/admin", ""},
		{http.MethodGet, "/entries/blocked-drain/private", ""},
		{http.MethodPut, "/entries/blocked-drain/private", `{"readme":"x"}`},
		{http.MethodPost, "/entries/blocked-drain/visibility", `{"visibility":"public"}`},
		{http.MethodGet, "/entries/blocked-drain/links", ""},
		{http.MethodPost, "/entries/blocked-drain/links", `{"label":"again"}`},
		{http.MethodGet, "/entries/blocked-drain/matches", ""},
		{http.MethodDelete, "/entries/blocked-drain", ""},
	}
	for _, d := range operatorDoors {
		if rec := do(t, operator, d[0], d[1], d[2]); rec.Code != http.StatusNotFound {
			t.Errorf("%s %s on a deleted entry answered %d %s; want 404, the same as a slug that never existed", d[0], d[1], rec.Code, rec.Body.String())
		}
	}
	anonDoors := [][3]string{
		{http.MethodGet, "/entries/blocked-drain", ""},
		{http.MethodGet, "/entries/blocked-drain?k=" + issued.Key, ""},
		{http.MethodPost, "/page-views", `{"page":"entry","slug":"blocked-drain"}`},
	}
	for _, d := range anonDoors {
		rec := do(t, anon, d[0], d[1], d[2])
		missing := do(t, anon, d[0], strings.ReplaceAll(d[1], "blocked-drain", "never-filed"), strings.ReplaceAll(d[2], "blocked-drain", "never-filed"))
		if rec.Code != http.StatusNotFound || rec.Code != missing.Code || rec.Body.String() != missing.Body.String() {
			t.Errorf("anonymous %s %s on a deleted entry answered %d %s; a never-filed slug answered %d %s; want both the same 404", d[0], d[1], rec.Code, rec.Body.String(), missing.Code, missing.Body.String())
		}
	}

	resend := do(t, operator, http.MethodPut, "/entries/blocked-drain", body)
	if resend.Code == http.StatusOK {
		var e struct {
			Visibility string `json:"visibility"`
			ApprovedAt string `json:"approved_at"`
		}
		if err := json.Unmarshal(resend.Body.Bytes(), &e); err != nil {
			t.Fatalf("re-sending a deleted slug answered a body that is not an entry: %v", err)
		}
		if e.Visibility != "pending" || e.ApprovedAt != "" {
			t.Errorf("re-sending the unchanged document of a deleted entry answered visibility %q approved_at %q; want a fresh pending entry with no approval carried across the delete", e.Visibility, e.ApprovedAt)
		}
		if rec := do(t, operator, http.MethodGet, "/entries/blocked-drain/private", ""); strings.Contains(rec.Body.String(), "resident-phone-marker") {
			t.Errorf("re-sending a deleted slug brought its private notes back: %s", rec.Body.String())
		}
	} else if resend.Code != http.StatusConflict {
		t.Errorf("re-sending a deleted slug answered %d %s; want either a fresh pending entry or a refusal", resend.Code, resend.Body.String())
	}
	if rec := do(t, anon, http.MethodGet, "/entries/blocked-drain?k="+issued.Key, ""); rec.Code != http.StatusNotFound {
		t.Errorf("the share key issued before the delete opened the slug after a re-send: %d %s; want 404", rec.Code, rec.Body.String())
	}
	if rec := do(t, anon, http.MethodGet, "/public/entries", ""); strings.Contains(rec.Body.String(), "blocked-drain") {
		t.Errorf("anonymous GET /public/entries lists the deleted slug after a re-send: %s", rec.Body.String())
	}
}
