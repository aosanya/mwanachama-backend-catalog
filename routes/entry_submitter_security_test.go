package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestCAT14_EntrySubmitterIsNeverTakenFromTheRequestBody(t *testing.T) {
	h, cm, tools := newMountedServer(t)

	const claimed = "someone-else-entirely"
	written := do(t, h, "PUT", "/entries/an-entry", `{"name":"Agency","doc":`+strconv.Quote(doc)+`,"submitted_by":"`+claimed+`"}`)
	if written.Code != http.StatusOK {
		t.Fatalf("precondition: writing an entry answered %d %s", written.Code, written.Body.String())
	}

	stored, err := cm.GetEntry(context.Background(), "an-entry")
	if err != nil {
		t.Fatalf("precondition: the written entry could not be read back: %v", err)
	}
	if stored.SubmittedBy != mountedCaller {
		t.Errorf("an entry written over REST recorded submitted_by %q; want the mount's caller %q", stored.SubmittedBy, mountedCaller)
	}

	upsert := toolNamed(t, tools, "catalog_entry_upsert")
	body := `{"slug":"another-entry","name":"Agency","doc":` + strconv.Quote(doc)
	_, err = upsert(context.Background(), json.RawMessage(body+`,"submitted_by":"`+claimed+`"}`))
	if err == nil || !strings.Contains(err.Error(), `"submitted_by"`) {
		t.Errorf("catalog_entry_upsert accepted a caller-supplied submitter; want a refusal naming \"submitted_by\", got err=%v", err)
	}
	if _, err := upsert(context.Background(), json.RawMessage(body+`}`)); err != nil {
		t.Fatalf("precondition: writing an entry over MCP without a submitter failed: %v", err)
	}
	stored, err = cm.GetEntry(context.Background(), "another-entry")
	if err != nil {
		t.Fatalf("precondition: the entry written over MCP could not be read back: %v", err)
	}
	if stored.SubmittedBy != mountedCaller {
		t.Errorf("an entry written over MCP recorded submitted_by %q; want the mount's caller %q", stored.SubmittedBy, mountedCaller)
	}
}
