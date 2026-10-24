package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestARaisedItemsSubjectIsCheckedAndListedThroughTheMux(t *testing.T) {
	gated, anon, cm := callerMount(t, "amina")
	holder, err := cm.CreateEntry(t.Context(), catalog.Entry{Slug: "holder-one", Name: "Holder one", Doc: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	if rec := sendTo(gated, http.MethodPost, "/raised-items", `{"title":"Dark road","subject_id":"no-such-entry"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown subject raise = %d %s", rec.Code, rec.Body)
	}
	if rec := sendTo(gated, http.MethodPost, "/raised-items", `{"title":"Dark road","subject_id":"`+holder.ID+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("known subject raise = %d %s", rec.Code, rec.Body)
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
