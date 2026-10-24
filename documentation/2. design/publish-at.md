# Publishing at a date

CAT62, from the member app's decision 290 (owner, 2026-10-22): an entry can
be set to go public at a date and time.

## The shape

`entry.publish_at` is an optional timestamp on the blueprint and on
`models.Entry`. It is RFC 3339 on the way in, any offset, and is stored in
UTC (`2030-01-01T03:00:00+03:00` is kept as `2030-01-01T00:00:00Z`), so the
text column orders the same way the clock does and compares directly with
the store's own `now`. Anything that does not parse is `ErrInvalidEntry`.
Empty means "as soon as it is public", which is every entry written before
the field existed.

`CreateEntry` and `UpdateEntry` carry it from the body. `UpsertEntry` does
not (decision 404, 2026-10-24): an editor who may write an entry but not
publish it must not move its go-live date, so an upsert creating an entry
stores none, and an upsert replacing one keeps the stored `publish_at`
whatever the body says. It does not touch the review —
`visibility` and `approved_at` still say whether the entry was approved, and
`publish_at` only says from when an approved entry may be read.

## What it hides

Every anonymous read treats an entry whose `publish_at` is still to come as
not there, answering exactly as for a missing slug:

- `OpenEntry` (`GET /entries/{slug}`) answers `ErrEntryNotFound`, whether
  the entry is public or private with a live key.
- `RecordPageView` reaches the entry through `OpenEntry`, so a view of it is
  refused the same way and never counted.
- `ListPublicEntries` leaves it out.

The operator's `GetEntry` and `ListEntries` still see it, because an operator
has to be able to find what is scheduled. Under `WithAudience` only staff do:
below staff those two read the way anonymous reads do (CAT131, see
[audience.md](audience.md)).

## Scheduling, one entry or a batch

Decision 404 puts scheduling under the publish action,
`catalog.entry.set_visibility`. The manager has the two calls:

- `SchedulePublish(ctx, slug, publishAt)` sets or clears (empty) one
  entry's `publish_at`.
- `SchedulePublishBatch(ctx, PublishSchedule{Slugs, PublishAt})` sets one
  time on every named entry in a single write. Every slug is looked up
  first; one naming no entry refuses the whole batch with `ErrEntryNotFound`
  wrapped with that slug (404, the message names it) and nothing is
  written. No slugs, or a time that does not parse, is `ErrInvalidEntry`. A
  slug named twice is scheduled once.

## What is left

- **The addresses.** Neither call is declared in `catalog.operations.json`
  yet. `dispatch.Parse` refuses two operations claiming one action ("both
  claim the action"), because an anonymous allowlist names actions and a
  shared action would open every operation carrying it. So an operation
  under `catalog.entry.set_visibility` beside `set_visibility` itself cannot
  be declared without either a change to shared's engine (letting gated
  operations share an action) or an action of its own that a host grants
  together with `set_visibility`. That choice is the owner's.
- `CreateEntry` and `UpdateEntry` still carry `publish_at` from the body;
  decision 404 named the upsert only.
