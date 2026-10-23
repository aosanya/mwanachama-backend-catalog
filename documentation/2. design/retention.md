# Retention that ends

CAT74, from the municipality template's retention wish (AGX14) and the member
app's decision 283: an entry should be able to have its keeping end, and be
disposed of when it does, rather than living until somebody remembers to
delete it.

## The shape

`entry.dispose_at` is an optional timestamp on the blueprint and on
`models.Entry`, beside `publish_at` and treated the same way: RFC 3339 on the
way in, any offset, stored in UTC so the text column orders as the clock
does; anything that does not parse is `ErrInvalidEntry`. Empty means "kept
until somebody deletes it", which is every entry written before the field
existed. `CreateEntry`, `UpsertEntry` and `UpdateEntry` carry it from the
body. `Entry.RetainedAt(now)` says whether an entry is still inside its
retention at a given instant.

## The sweep

`DisposeExpired(ctx, now time.Time) ([]string, error)` disposes of every
non-deleted entry whose `dispose_at` is at or before `now`, and answers the
slugs it disposed, in slug order. The boundary is inclusive: an entry due at
`T` goes at a sweep at `T`.

**The module never schedules itself.** The host calls the sweep on whatever
schedule it keeps, and passes the time; that is what lets the tests step a
clock across a month and a year without waiting for either.

Each disposal is a conditional write on `deleted = false` and the due time,
so two sweeps racing dispose of an entry once and only one of them reports
it. A sweep that finds nothing due answers an empty list, and a second sweep
at the same time answers nothing, because what the first disposed is already
gone.

## What disposing means

Disposing is the existing soft delete, exactly what `DeleteEntry` does: the
row is marked `deleted`, its `updated_at` moves to the sweep's time, and
every live share key on it is retired. Every read then answers as for a
missing slug. Nothing is erased from the store.

That is the conservative choice, not the settled one. A retention schedule
that must destroy a record (a hard delete, or a scrub of `doc`) is a stronger
act and is left for the owner to decide; see the open questions below.

## Not built, and open

- **Hard delete or soft delete.** Whether disposal must erase the row, or
  blank its document, rather than mark it deleted.
- **A retention class.** The row asked for "a retention class per object".
  What is built is a per-entry instant. A named class with a period
  (`seven_years`), defaulted by the domain spec and turned into a
  `dispose_at` on write, needs a duration type the spec engine does not have
  yet, and a decision on what the period counts from (creation, approval or
  publication).
- **Other objects.** Only the entry carries a retention. Raised items,
  records and the rest do not.
- **No REST or MCP address.** The sweep is a Go call the host makes on its
  schedule; nothing outside the host can trigger it.
