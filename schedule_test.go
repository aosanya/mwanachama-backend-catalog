package mwanachamacatalog_test

import (
	"errors"
	"strings"
	"testing"

	catalog "github.com/aosanya/mwanachama-backend-catalog"
)

func TestSchedulePublishBatchSetsEveryNamedEntry(t *testing.T) {
	cm, ctx := newManager(t)
	createScheduled(t, cm, "batch-a", "")
	createScheduled(t, cm, "batch-b", "")

	out, err := cm.SchedulePublishBatch(ctx, catalog.PublishSchedule{Slugs: []string{"batch-a", "batch-b", "batch-a"}, PublishAt: "2030-01-01T03:00:00+03:00"})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("batch answered %d entries, want 2", len(out))
	}
	for _, slug := range []string{"batch-a", "batch-b"} {
		e, err := cm.GetEntry(ctx, slug)
		if err != nil {
			t.Fatalf("read %s: %v", slug, err)
		}
		if e.PublishAt != "2030-01-01T00:00:00Z" {
			t.Errorf("%s publish_at = %q, want 2030-01-01T00:00:00Z", slug, e.PublishAt)
		}
		if _, err := cm.OpenEntry(ctx, slug, ""); !errors.Is(err, catalog.ErrEntryNotFound) {
			t.Errorf("anonymous open of scheduled %s: err = %v, want not found", slug, err)
		}
	}

	if _, err := cm.SchedulePublishBatch(ctx, catalog.PublishSchedule{Slugs: []string{"batch-a"}}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if _, err := cm.OpenEntry(ctx, "batch-a", ""); err != nil {
		t.Errorf("a cleared schedule still hides batch-a: %v", err)
	}
}

func TestSchedulePublishBatchNamingAMissingSlugWritesNothing(t *testing.T) {
	cm, ctx := newManager(t)
	createScheduled(t, cm, "kept", "")

	_, err := cm.SchedulePublishBatch(ctx, catalog.PublishSchedule{Slugs: []string{"kept", "nowhere"}, PublishAt: "2030-01-01T00:00:00Z"})
	if !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Fatalf("err = %v, want ErrEntryNotFound", err)
	}
	if !strings.Contains(err.Error(), "nowhere") {
		t.Errorf("the refusal %q does not name the missing slug", err)
	}
	e, err := cm.GetEntry(ctx, "kept")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if e.PublishAt != "" {
		t.Errorf("a refused batch wrote publish_at %q on kept", e.PublishAt)
	}
}

func TestSchedulePublishBatchRefusesNoSlugsAndABadTime(t *testing.T) {
	cm, ctx := newManager(t)
	createScheduled(t, cm, "one", "")
	if _, err := cm.SchedulePublishBatch(ctx, catalog.PublishSchedule{PublishAt: "2030-01-01T00:00:00Z"}); !errors.Is(err, catalog.ErrInvalidEntry) {
		t.Errorf("no slugs: err = %v, want ErrInvalidEntry", err)
	}
	if _, err := cm.SchedulePublishBatch(ctx, catalog.PublishSchedule{Slugs: []string{"one"}, PublishAt: "tomorrow"}); !errors.Is(err, catalog.ErrInvalidEntry) {
		t.Errorf("bad time: err = %v, want ErrInvalidEntry", err)
	}
}

func TestSchedulePublishSetsOneEntry(t *testing.T) {
	cm, ctx := newManager(t)
	createScheduled(t, cm, "single", "")
	e, err := cm.SchedulePublish(ctx, "single", "2030-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if e.PublishAt != "2030-01-01T00:00:00Z" {
		t.Errorf("publish_at = %q", e.PublishAt)
	}
	if _, err := cm.SchedulePublish(ctx, "absent", "2030-01-01T00:00:00Z"); !errors.Is(err, catalog.ErrEntryNotFound) {
		t.Errorf("absent: err = %v, want not found", err)
	}
}
