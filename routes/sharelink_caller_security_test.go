package routes_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
	"github.com/aosanya/mwanachama-backend-catalog/routes"
)

func TestCAT12_ShareLinkToolRefusesAnIssuerArgumentAndRecordsTheCaller(t *testing.T) {
	_, cm, ctx := newServer(t)
	seed(t, cm, ctx, "private-one", catalog.VisibilityPrivate)

	var issue routes.Tool
	for _, tool := range routes.ToolsFor(cm, routes.Mount{Caller: func(context.Context) string { return "operator-a" }}) {
		if tool.Action == "catalog.share_link.issue" {
			issue = tool
		}
	}
	if issue.Invoke == nil {
		t.Fatal("precondition: no tool carries catalog.share_link.issue")
	}
	if strings.Contains(string(issue.InputSchema), `"by"`) {
		t.Errorf("the share link tool's schema offers an issuer argument: %s", issue.InputSchema)
	}

	if _, err := issue.Invoke(ctx, json.RawMessage(`{"slug":"private-one","label":"x","by":"someone-else"}`)); err == nil {
		t.Error("a tool call naming its own issuer was accepted; it should be refused as an unknown argument")
	}

	if _, err := issue.Invoke(ctx, json.RawMessage(`{"slug":"private-one","label":"x"}`)); err != nil {
		t.Fatalf("precondition: issuing without an issuer argument failed: %v", err)
	}
	links, err := cm.ListShareLinks(ctx, "private-one")
	if err != nil || len(links) != 1 {
		t.Fatalf("precondition: expected one stored link, got %d (%v)", len(links), err)
	}
	if links[0].CreatedBy != "operator-a" {
		t.Errorf("the stored issuer is %q; it should be the mount's caller %q", links[0].CreatedBy, "operator-a")
	}
}
