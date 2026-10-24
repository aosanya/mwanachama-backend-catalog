# Scope figures

CAT72. A register counts rows; a coverage figure ("how many of the people in
this place have we heard from") also needs a denominator, and the
denominator is an outside estimate this mount cannot produce. `scope_figure`
is where that estimate lives, so it is carried beside the count rather than
typed into a report by hand.

## Shape

| Field | Owner | Meaning |
| --- | --- | --- |
| `id` | store | minted on record; a body's `id` is ignored |
| `scope_id` | caller's body | the scope the figure is about, opaque here |
| `value` | caller's body | a whole number, never negative |
| `as_of` | caller's body | when the figure was true, RFC 3339, stored in UTC |
| `source` | caller's body | where it came from, as a reader would cite it |
| `recorded_by` | mount | bound `from: caller`, never read from the request |
| `created_at` | store | when it was recorded |

A role a domain spec may leave unfilled: a domain with no `scope_figure`
object answers `ErrFiguresNotKept` (404) on all three operations.

## Append-only, newest `as_of` wins

A figure is never edited. A new estimate is a new row, and the current
figure is the one with the latest `as_of` (ties broken by `created_at`, then
`id`), not the one recorded last: entering an older census after a newer
survey must not make the older one current. `as_of` is normalised to UTC
before it is stored, because timestamps compare as text.

## Operations

- `POST /scope-figures` — `catalog.scope_figure.record`
- `GET /scope-figures/{scope_id}` — `catalog.scope_figure.read`, the current figure
- `GET /scope-figures/{scope_id}/history` — `catalog.scope_figure.list`, newest first, at most 100

All three are gated; none is on `AnonymousActions`. Who may record a figure
is the mount's grant to decide, not this module's.

## Domain words

The module says *scope* and *figure*. The municipality's "ward" and
"population" are in `spec/examples/agency.catalog.json`'s description only.
The coverage division itself is left to the reader of both numbers; this
module stores the denominator and does no arithmetic with it.
