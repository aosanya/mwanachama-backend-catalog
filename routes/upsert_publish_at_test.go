package routes_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
)

func TestUpsertThroughTheMuxNeitherSetsNorMovesPublishAt(t *testing.T) {
	h, cm, _ := newMountedServer(t)
	ctx := context.Background()

	if rec := do(t, h, http.MethodPut, "/entries/dated", `{"name":"Dated","publish_at":"2031-01-01T00:00:00Z","doc":`+strconv.Quote(doc)+`}`); rec.Code != http.StatusOK {
		t.Fatalf("precondition: upsert answered %d %s", rec.Code, rec.Body.String())
	}
	stored, err := cm.GetEntry(ctx, "dated")
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if stored.PublishAt != "" {
		t.Errorf("a creating upsert stored publish_at %q; want none", stored.PublishAt)
	}

	if _, err := cm.SchedulePublish(ctx, "dated", "2030-06-01T00:00:00Z"); err != nil {
		t.Fatalf("precondition: schedule: %v", err)
	}
	for _, body := range []string{
		`{"name":"Dated","publish_at":"2001-01-01T00:00:00Z","doc":` + strconv.Quote(doc) + `}`,
		`{"name":"Dated","publish_at":"","doc":` + strconv.Quote(doc) + `}`,
		`{"name":"Dated","doc":` + strconv.Quote(doc) + `}`,
	} {
		if rec := do(t, h, http.MethodPut, "/entries/dated", body); rec.Code != http.StatusOK {
			t.Fatalf("upsert answered %d %s", rec.Code, rec.Body.String())
		}
		stored, err := cm.GetEntry(ctx, "dated")
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		if stored.PublishAt != "2030-06-01T00:00:00Z" {
			t.Errorf("an upsert with body %s left publish_at %q; want the scheduled 2030-06-01T00:00:00Z kept", body, stored.PublishAt)
		}
	}
}
