package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT12_OpenHole_ShareLinkIssuerIsTakenFromTheRequestBody(t *testing.T) {
	h, cm, ctx := newServer(t)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	const claimed = "someone-else-entirely"
	issued := do(t, h, "POST", "/entries/private-one/links", `{"label":"for a reviewer","by":"`+claimed+`"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("precondition: issuing a share link answered %d %s", issued.Code, issued.Body.String())
	}

	listed := do(t, h, "GET", "/entries/private-one/links", "")
	var links []catalog.ShareLink
	if err := json.Unmarshal(listed.Body.Bytes(), &links); err != nil || len(links) != 1 {
		t.Fatalf("precondition: expected one stored link, got %d %s", listed.Code, listed.Body.String())
	}
	if links[0].CreatedBy == claimed {
		t.Errorf("CAT12 (P2) open hole: the stored share link's created_by is %q, the value the request body claimed; "+
			"it should record the caller that actually issued the key, never a value the caller supplied", links[0].CreatedBy)
	}
}
