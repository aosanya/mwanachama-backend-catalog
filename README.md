# mwanachama-backend-catalog

A catalog of documents with a review state: things are proposed, built,
reviewed, published or shared under a key, and counted.

A Go library backed by Postgres through GORM. There is no service to run —
`Migrate(db, tables)` creates the tables and `NewCatalogManager(db, tables)`
returns the manager the rest of your code calls.

## It names no domain, and that is the point

**Nothing in this module knows what a document is about.** It was written for
an agency storefront and is expected to run a library, a course catalog or a
product catalog without a line changing.

The rule is one sentence:

> A word that means something in one domain and nothing in another does not
> belong in this module.

"Sector" fails it — a library has genres. "Entry" passes it.

Three things already failed that test and were moved out. They are worth
knowing about, because they are the shape every future addition should be
checked against:

| Was | Why it failed | Where it went |
| --- | ------------- | ------------- |
| `Entry.Sector`, `Entry.Archetype` — columns | agency words | inside `Doc`, filtered by JSON path |
| `Suggestion.Sector` — a column | same | inside `Proposal` |
| `PageKind` values `"home"` / `"agency"` | the second is an agency word | `"index"` / `"entry"` |

What survives beside `Doc` is only what means the same everywhere: a slug, a
name, a summary, a review state, timestamps.

### Filtering on a domain's own words

A domain's fields live in the document and are filtered by path, so the
module never learns either vocabulary:

```go
// an agency catalog
cm.ListEntries(ctx, catalog.ListFilter{
    Docs: []catalog.DocFilter{{Path: "_template.sector", Value: "health"}},
})

// a library, same store, same code
cm.ListEntries(ctx, catalog.ListFilter{
    Docs: []catalog.DocFilter{{Path: "subject", Value: "history"}},
})
```

The path is validated against a strict alphabet before it reaches SQL — see
"The document column" below.

### One store, two domains

| catalog | an agency shop | a library |
| ------- | -------------- | --------- |
| `Entry` + `Doc` | a mined archetype | a title's record |
| `EntryPrivate` | readme, sales, module configs | acquisition notes, supplier terms |
| `ShareLink` | preview an unpublished agency | preview an uncatalogued title |
| `Suggestion` | an archetype worth mining | a patron purchase request |
| `SuggestionMatch` | the agency that answered it | the copy acquired |
| `MiningRun` | a mine-agencies run | a MARC import run |
| `PageView` | views per agency | views per title |

**When adding anything here, the test is whether a librarian would recognise
it.** If it needs the word "agency" to make sense, it belongs in the document
or in the caller.

## Why it is its own module

It replaces 11 hand-written tables and ~33 `security definer` functions that
lived only in `mwanachama-website/supabase/migrations/`. Two existing modules
were measured against it first and neither could hold it:

- **forms** models the lifecycle *better* than that SQL does — `PublicLink`
  keeps key, status and `retired_at` where the SQL overwrote a single hash,
  and `Approval` is a real submit/withdraw/approve/refuse. But a Form's
  content is its Questions: there is no JSON, blob or payload field in any of
  its models, and the catalog is 21 MB of documents.
- **assetmanager** maps a slug onto `Code` and a grouping onto `Category`
  cleanly, then has nowhere for a 34 KB document but `AttributesJSON`, a
  plain `text` column its own comment calls a v1 stand-in — and carries
  `Movement` and `Hold`, which a catalog never uses.

`ShareLink` and the review state are modelled on forms's shapes rather than
reinvented.

## The document column

`Entry.Doc` is `jsonb` on Postgres and `text` everywhere else, chosen per
dialect by `gormstore.JSONDoc`. Filtering inside the document happens in the
database — `doc #>> '{a,b}'` on Postgres, `json_extract(doc, '$.a.b')` on
SQLite, which has JSON1.

Neither dialect takes the path as a bound parameter, so
`models.DocPathPattern` restricts a path to dotted segments of letters,
digits, underscore and hyphen. Anything else is refused with
`ErrInvalidDocPath` rather than escaped.

## What it enforces

- Every Entry lands `pending`, whatever visibility the caller asked for.
- Changing a document returns it to `pending` and clears its approval, so
  nothing unreviewed is readable. Re-sending an *unchanged* document does
  not, which is what lets a run re-visit what it already published.
- A missing slug, a pending Entry, and a private Entry opened with a wrong or
  absent key are all the same `ErrEntryNotFound` — the catalog cannot be
  enumerated by probing.
- A share key is stored only as its SHA-256. The raw key is returned once, at
  issue.
- Leaving `private` retires every live key, so making an Entry private a
  second time does not reopen it to whoever held the old one.
- A listing never carries `Doc`.
- A proposal is deduped at filing against both the catalog and the existing
  proposals.
- A build attempt must say what is left; the tenth stalls the proposal, and
  approving it again clears the count for another ten.
- Only fulfilling a proposal marks it `built`, and only against an Entry that
  exists.
- An Entry page view counts only against a slug that exists.

## What it deliberately leaves out

**Who may do any of this.** The SQL this replaces keyed its admin table off
Supabase Auth's `auth.users`. This module has no auth model, matching
`mwanachama-backend-assetmanager`: a route still needs a capability gate
wrapped around it by whatever mounts it. The grant table belongs in
`mwanachama-backend-auth` — see
`developer/documentation/2. design/architecture-spec-driven-modules.md` §5.

## Status

Implemented and tested against SQLite: Entry CRUD with visibility and the
public read, the private parts, share links, the backlog (suggestions,
comments, matches, attempts and the stall), run reports, settings and page
views — nine tables.

`routes/` is **provisional and should not be copied.** It hand-writes twelve
handlers, which is the shape the spec-driven dispatcher replaces; its tests
are kept as the acceptance criteria that dispatcher must satisfy unchanged.

Not yet built: the dispatcher, the Postgres integration test covering the
`jsonb` reads, and running `cmd/import` against a real database. Tracked as
W13 and W14 on `mwanachama-website`'s board.

## Test

```sh
go test ./...        # in-memory SQLite, no database needed
make test-pg         # against a real Postgres, with POSTGRES_URL set
```

## Licence

Apache-2.0.
