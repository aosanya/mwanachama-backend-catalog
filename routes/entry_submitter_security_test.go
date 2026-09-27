package routes_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
)

func TestCAT14_OpenHole_EntrySubmitterIsTakenFromTheRequestBody(t *testing.T) {
	h, cm, _ := newMountedServer(t)

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
		t.Errorf("CAT14 (P1) open hole: the stored entry revision's submitted_by is %q, the value the request body claimed; "+
			"it should be %q, the caller the mount supplied, never a value the caller supplied", stored.SubmittedBy, mountedCaller)
	}
}
