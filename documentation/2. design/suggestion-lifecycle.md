# Suggestion lifecycle

How a suggestion moves, and what keeps one caller from undoing another.

## Statuses

`proposed`, `approved`, `rejected`, `built`, `stalled`. Only fulfilling sets
`built`; only the tenth failed attempt sets `stalled`; deciding sets the other
three.

## Writes are conditional on the status they read

Every write that depends on the status a function read carries that status in
its `where`, and a write that matches no row answers `ErrSuggestionStatus`.

| Call | Lands only while the suggestion is |
| --- | --- |
| `DecideSuggestion` | not `built` |
| `FulfilSuggestion` | `approved` or `built` |
| `RecordSuggestionAttempt` | `approved`, with the attempt count it read |

A reviewer's reject therefore survives a fulfil or a failed-attempt report in
flight beside it (CAT19, CAT21). `built` is terminal for deciding (CAT47): an
entry and a match row already answer the suggestion.

`RecordSuggestionAttempt` re-reads and retries when another report moved the
count first, so concurrent failures are each counted and the tenth stalls.

## Priority

`DecideSuggestion` takes `*int`. Absent keeps the stored priority; any integer,
zero and negative included, replaces it (CAT38). Lowest is first.

## Matches

`AttachSuggestion` accepts `proposed`, `approved` and `built`, and refuses
`rejected` and `stalled`. `ListEntryMatches` and `ListSuggestionMatches` leave
out matches of suggestions in any other status, so rejecting an attached
arrival drops it from the count and re-approving brings it back (CAT31).

`upsertMatch` updates first, inserts when nothing was updated, and updates
again when the insert loses a race on the composite key (CAT43).

## The anonymous door

Routes served through `PublicRoutesFor` run with `catalog.AsAnonymous` on the
request context. `FileSuggestion` under that context:

- ignores `priority`, `source` and `run_ref` from the body (CAT29), because
  they are the moderator's fields once filing is open to the public;
- never answers differently for a slug that is a hidden entry or another
  filer's suggestion (CAT30). A colliding report is stored under the slug plus
  a short random suffix, so the moderator still sees it, and the answer carries
  the slug the filer asked for.

Callers without the anonymous context (an operator, an in-process caller) keep
the strict dedup and receive `ErrSuggestionDuplicate`.

Not covered: the extra store read a colliding slug costs is a timing
difference.
