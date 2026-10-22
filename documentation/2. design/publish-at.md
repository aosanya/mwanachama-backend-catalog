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

It is written like every other field: `CreateEntry`, `UpsertEntry` and
`UpdateEntry` carry it from the body. It does not touch the review —
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
has to be able to find what is scheduled.

## What is left

- **A batch.** Setting one `publish_at` on several entries in one call is
  not built. It needs an operation of its own taking a list of slugs, and a
  decision on what a batch does when one slug is missing.
- **Who may set it.** It rides the entry's existing write actions, so
  whoever may write an entry may schedule it. Whether scheduling should be a
  separate action is the owner's question.
