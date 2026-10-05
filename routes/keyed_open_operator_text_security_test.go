package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestKeyedAnonymousOpenCarriesNeitherTheLinkLabelNorThePrivateNotes(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	const notes = "operator-only-readme-marker"
	const label = "label-naming-the-recipient-marker"

	if rec := do(t, h, http.MethodPut, "/entries/private-one/private", `{"readme":"`+notes+`","rank":3}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: writing private notes answered %d %s", rec.Code, rec.Body.String())
	}
	issued := do(t, h, http.MethodPost, "/entries/private-one/links", `{"label":"`+label+`"}`)
	if issued.Code != http.StatusCreated || !strings.Contains(issued.Body.String(), label) {
		t.Fatalf("precondition: issuing a labelled share link answered %d %s", issued.Code, issued.Body.String())
	}
	var key struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(issued.Body.Bytes(), &key); err != nil || key.Key == "" {
		t.Fatalf("precondition: no raw key in %s", issued.Body.String())
	}

	opened := do(t, h, http.MethodGet, "/entries/private-one?k="+key.Key, "")
	if opened.Code != http.StatusOK {
		t.Fatalf("precondition: the keyed open answered %d %s", opened.Code, opened.Body.String())
	}
	for _, operatorText := range []string{notes, label} {
		if strings.Contains(opened.Body.String(), operatorText) {
			t.Errorf("a keyed anonymous open carried operator-only text %q: %s", operatorText, opened.Body.String())
		}
	}
}
