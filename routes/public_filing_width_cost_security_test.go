package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func wideProposalBody(slug string, keys int) string {
	var b strings.Builder
	b.WriteString("{")
	for i := 0; i < keys; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `\"k%d\":1`, i)
	}
	b.WriteString("}")
	return `{"slug":"` + slug + `","title":"t","proposal":"` + b.String() + `"}`
}

func TestAnonymousWideProposalAndLongSearchStayLinearAndLeaveTheQueueReadable(t *testing.T) {
	operator, anon, _ := municipalityWard(t)

	started := time.Now()
	wide := do(t, anon, "POST", "/suggestions", wideProposalBody("wide-filing", 50000))
	filing := time.Since(started)
	if wide.Code != http.StatusCreated {
		t.Fatalf("precondition: an anonymous filing with a 50,000-key proposal answered %d %s, want 201", wide.Code, wide.Body.String())
	}
	if filing > 5*time.Second {
		t.Errorf("an anonymous 50,000-key proposal took %v to file, want well under 5s (40ms when pinned)", filing)
	}

	queue := do(t, operator, "GET", "/suggestions", "")
	var rows []catalog.Suggestion
	if err := json.Unmarshal(queue.Body.Bytes(), &rows); err != nil || queue.Code != http.StatusOK {
		t.Fatalf("after a wide anonymous filing the moderator queue answered %d (%v), want a readable 200", queue.Code, err)
	}
	found := false
	for _, r := range rows {
		if r.Slug != "wide-filing" {
			continue
		}
		found = true
		var v map[string]any
		if err := json.Unmarshal([]byte(r.Proposal), &v); err != nil || len(v) != 50000 {
			t.Errorf("the wide proposal reads back as %d keys (%v), want all 50,000", len(v), err)
		}
	}
	if !found {
		t.Errorf("the moderator queue does not hold the wide anonymous filing")
	}

	started = time.Now()
	search := do(t, anon, "GET", "/public/entries?search="+strings.Repeat("a", 100000), "")
	searching := time.Since(started)
	if search.Code != http.StatusOK {
		t.Errorf("an anonymous 100,000-character search answered %d %s, want 200", search.Code, search.Body.String())
	}
	if searching > 5*time.Second {
		t.Errorf("an anonymous 100,000-character search took %v, want well under 5s (1ms when pinned)", searching)
	}
}
