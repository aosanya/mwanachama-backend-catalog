package routes_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestCAT40_OpenHole_AnonymousReadCarriesTheWritersIdentity(t *testing.T) {
	h, _, _ := newMountedServer(t)

	written := do(t, h, "PUT", "/entries/a-public-entry", `{"name":"Agency","doc":`+strconv.Quote(doc)+`}`)
	if written.Code != http.StatusOK || !strings.Contains(written.Body.String(), mountedCaller) {
		t.Fatalf("precondition: the operator's write should record the mount's caller; got %d %s", written.Code, written.Body.String())
	}
	published := do(t, h, "POST", "/entries/a-public-entry/visibility", `{"visibility":"public"}`)
	if published.Code != http.StatusOK {
		t.Fatalf("precondition: publishing the entry answered %d %s", published.Code, published.Body.String())
	}

	for _, read := range []string{"/entries/a-public-entry", "/public/entries"} {
		got := do(t, h, "GET", read, "")
		if got.Code != http.StatusOK || !strings.Contains(got.Body.String(), "a-public-entry") {
			t.Fatalf("precondition: an anonymous GET %s should serve the public entry; got %d %s", read, got.Code, got.Body.String())
		}
		if strings.Contains(got.Body.String(), mountedCaller) {
			t.Errorf("CAT40 (P3): an anonymous GET %s handed the reader the writing operator's identity %q in %s; want the anonymous view to carry no submitted_by", read, mountedCaller, got.Body.String())
		}
	}
}
