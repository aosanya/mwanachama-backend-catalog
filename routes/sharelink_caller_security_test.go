package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestCAT12_AMountedShareLinkRecordsTheMountsCallerAndMCPRefusesAnIssuer(t *testing.T) {
	h, cm, tools := newMountedServer(t)
	seed(t, cm, context.Background(), "private-one", catalog.VisibilityPrivate)

	issued := do(t, h, "POST", "/entries/private-one/links", `{"label":"rest","by":"someone-else-entirely"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("precondition: issuing a share link answered %d %s", issued.Code, issued.Body.String())
	}
	var links []catalog.ShareLink
	listed := do(t, h, "GET", "/entries/private-one/links", "")
	if err := json.Unmarshal(listed.Body.Bytes(), &links); err != nil || len(links) != 1 {
		t.Fatalf("precondition: expected one stored link, got %d %s", listed.Code, listed.Body.String())
	}
	if links[0].CreatedBy != mountedCaller {
		t.Errorf("a share link issued over REST recorded created_by %q; want the mount's caller %q", links[0].CreatedBy, mountedCaller)
	}

	var issue func(context.Context, json.RawMessage) (any, error)
	for _, tl := range tools {
		if tl.Name == "catalog_share_link_issue" {
			issue = tl.Invoke
		}
	}
	if issue == nil {
		t.Fatal("precondition: no catalog_share_link_issue tool is built")
	}
	_, err := issue(context.Background(), json.RawMessage(`{"slug":"private-one","label":"mcp","by":"someone-else-entirely"}`))
	if err == nil || !strings.Contains(err.Error(), `"by"`) {
		t.Errorf("catalog_share_link_issue accepted a caller-supplied issuer; want a refusal naming \"by\", got err=%v", err)
	}
	if _, err := issue(context.Background(), json.RawMessage(`{"slug":"private-one","label":"mcp"}`)); err != nil {
		t.Fatalf("precondition: issuing over MCP without an issuer failed: %v", err)
	}
	links = nil
	listed = do(t, h, "GET", "/entries/private-one/links", "")
	if err := json.Unmarshal(listed.Body.Bytes(), &links); err != nil || len(links) != 2 {
		t.Fatalf("expected two stored links after the MCP issue, got %d %s", listed.Code, listed.Body.String())
	}
	for _, l := range links {
		if l.CreatedBy != mountedCaller {
			t.Errorf("share link %q recorded created_by %q; want the mount's caller %q", l.Label, l.CreatedBy, mountedCaller)
		}
	}
}
