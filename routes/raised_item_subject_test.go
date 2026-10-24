package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestAResidentRaiseDropsItsSubjectAndAFiledOneIsListedBySubjectThroughTheMux(t *testing.T) {
	gated, anon, cm := callerMount(t, "amina")
	holder, err := cm.CreateEntry(t.Context(), catalog.Entry{Slug: "holder-one", Name: "Holder one", Doc: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	if rec := sendTo(gated, http.MethodPost, "/raised-items", `{"title":"Streetlight","subject_id":"`+holder.ID+`"}`); rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &struct{}{}) != nil {
		t.Fatalf("resident raise naming a subject = %d %s", rec.Code, rec.Body)
	} else {
		var raised catalog.RaisedItem
		if json.Unmarshal(rec.Body.Bytes(), &raised) != nil || raised.SubjectID != "" {
			t.Fatalf("resident raise kept subject %q", raised.SubjectID)
		}
	}
	if _, err := cm.FileRaisedItem(t.Context(), catalog.RaisedItem{Title: "Dark road", RaisedBy: "amina", FiledBy: "clerk-1", Channel: "phone", SubjectID: holder.ID}); err != nil {
		t.Fatal(err)
	}
	if rec := sendTo(gated, http.MethodPost, "/raised-items", `{"title":"Pothole"}`); rec.Code != http.StatusCreated {
		t.Fatalf("no subject raise = %d %s", rec.Code, rec.Body)
	}
	rec := sendTo(gated, http.MethodGet, "/raised-items/by-subject?subject_id="+holder.ID, "")
	var listed []catalog.RaisedItem
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &listed) != nil || len(listed) != 1 || listed[0].Title != "Dark road" {
		t.Fatalf("by subject = %d %s", rec.Code, rec.Body)
	}
	if rec := sendTo(anon, http.MethodGet, "/raised-items/by-subject?subject_id="+holder.ID, ""); rec.Code == http.StatusOK {
		t.Fatalf("anonymous by subject = %d %s", rec.Code, rec.Body)
	}
}
