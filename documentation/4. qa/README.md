# QA

Test coverage and results for `mwanachama-backend-catalog`. The tests live
beside the code they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
make test-pg         # -tags=integration, with POSTGRES_URL set
```

As of 2026-09-26 (after `spec/` moved to `mwanachama-backend-shared`, S18,
and this repo's own store/blueprint/manager were refactored onto it,
commit 37e0ae7): 81 test functions offline, all green — `entry_test.go`
(14), `suggestion_test.go` (12), `routes/tools_test.go` (10), `run_test.go`
(8), `examples_test.go` (8), `routes/routes_test.go` (7), `sharelink_test.go`
(5), `private_test.go` (5), `store_test.go` (4), `domain_agnostic_test.go`
(3), `blueprint_test.go` (3), `validate_test.go` (2), plus 7 more in
`postgres_integration_test.go` that need a database. `spec/` now holds only
`examples/` — its own loader/validator/migrator tests moved with the code to
`mwanachama-backend-shared/spec`'s own suite. SQLite in memory is the
expected way to verify a change here — reach for a real Postgres only for
what the section below lists.

## What is guarded, and how

- **The behaviours in [CLAUDE.md](../../CLAUDE.md)** each have a test, and
  the first two were mutation-checked rather than trusted: leaking a pending
  entry from the identical-404 refusal fails the test, and turning the stall
  counter's `>=` into `>` fails it.
- **The domain-agnostic rule** is enforced, not asserted:
  `domain_agnostic_test.go` parses `models/`, `routes/` and the root package
  and fails on any identifier carrying a domain word, with a second test over
  every stored enum value — because a stored value outlives a rename. It was
  mutation-checked by reintroducing `Entry.Sector`. (`gormstore/` was
  deleted 2026-09-23, CAT1 — see CLAUDE.md's "What is superseded".)
- **The spec's refusals** — bad names, an underscore in a segment, a
  duplicate role, identifiers the database would merge past 63 bytes, a
  required field carrying a default, unknown JSON keys — are each their own
  test in `spec/spec_test.go`. The truncation and budget tests were
  mutation-checked by widening `MaxIdentifier` and by restoring the looser
  pattern on the segments.
- **Two domains in one database**: `TestTwoDomainsCoexist` migrates both
  example specs into one database; `TestMigrateIsIdempotent` runs a migration
  twice.
- **`routes/routes_test.go`** is kept deliberately as the acceptance test the
  W14 dispatcher must pass **unchanged**, even though the package it
  currently exercises is superseded.

## What only a real Postgres covers

`postgres_integration_test.go` is tagged `integration` and skips unless
`POSTGRES_URL` is set, so `go test ./...` stays offline. Seven tests, last
run green against the live `mwanachama-2` project (PostgreSQL 17.6):

- **`doc` is really `jsonb`** there and `text` on SQLite, read back off
  `information_schema` rather than assumed.
- **A document filter goes through `doc #>> '{_template,sector}'`** against a
  real jsonb column, the operator SQLite never sees.
- **Every declared path index is accepted** and appears in `pg_indexes` — and
  the companion test removes the inner parentheses from the emitted DDL and
  asserts Postgres *rejects* it. That is the database mutation-checking the
  double parens, which `TestExpressionIndexesAreDoubleParenthesised` can only
  assert about a string.
- **Identifier truncation at 63 bytes is silent**: two names agreeing that far
  are created as one relation, carrying only the first one's columns. This is
  the premise `spec.Validate` refuses on, and until now it was assumed.
- **A document survives jsonb semantically** but not byte-for-byte, which is
  what the review comparison had to learn.

Each run mints a random instance sized to the spec's own identifier budget,
so its tables are `pg<hex>_catalog_*` and are dropped on cleanup.

**That override is load-bearing, not tidiness.** Since 2026-09-23 the live
`mwanachama-2` database holds `agency_catalog_*` — 634 entries, seeded from
the spec, with the older bare `agency_*` tables dropped after a row-level
parity check. Those are exactly the names
`spec/examples/agency.catalog.json` emits unmodified, at instance `agency`
and module `catalog`. So anything that loads the shipped spec as-is and
points at that database is pointed at production. Override the instance, and
never re-seed or truncate it: ten rows from a 2026-09-21 mine-agencies rescue
run and our own `mwanachama-wakala` agency exist there and in no folder on
disk.

Three bugs so far were found by a real database rather than by the SQLite
tests — the missing inner parentheses, a slug beginning with a digit being
dropped by an identifier pattern, and the byte-wise document comparison in
`UpsertEntry` that jsonb's normalisation turned into a silent un-publish
(CAT2). That is the standing argument for keeping this suite runnable.

`cmd/import` has still never been run against a live database; `cmd/seed`'s
SQL path has (623 agencies into `mwanachama-2`).

## First live probe through a real mount (2026-09-26)

Until this date, this module's routes had only ever been exercised as a bare
library (`CatalogManager` called directly, as CAT10's own row above does) —
`mwanachama-wakala-api`'s mount (`buildCatalog`, `cmd/server/catalog.go`)
never calls `spec.Migrate`, so a freshly created Postgres database 500s on
every `/catalog/...` request until the spec's DDL is applied by hand (see
that repo's own `documentation/4. qa/README.md`). The 2026-09-26 scheduled
security sweep did that by hand once (a throwaway local Postgres, `cmd/ddl`
piped to `psql`) and drove the real router for the first time. Confirmed
live: `GET /entries/{slug}` (anonymous) refuses a never-registered slug, a
never-published (pending) Entry's own slug, and a private Entry's slug with
byte-identical `{"error":"entry not found"}` 404 bodies — the flagship
"Behaviours to preserve" invariant in CLAUDE.md, previously only proven at
the library level. Also confirmed: every operator route (`PUT`/`DELETE
/entries/{slug}`, `.../visibility`, `.../admin`) is reachable by any
authenticated caller and by none anonymously, and — because this module
"names no domain" and therefore has no submitter/ownership check of its own
— any two authenticated operators can fully read, overwrite, publish or
delete each other's Entries. That last part is not a defect: it matches
`mwanachama-wakala-api`'s own `documentation/2. design/tenancy-and-identity.md`
verbatim ("The catalog is outside it... gated on being a user of the
project, not on membership of an Instance... If the catalog ever needs its
own operator/reader split, it needs its own rule").

## Red by design (security sweep, 2026-09-27)

**None. Both reds this section carried were cleared 2026-09-27**, and each
test stays in place as the regression guard for its fix, with the board id
kept and the `_OpenHole_` infix dropped:

- `routes/pageview_security_test.go` —
  `TestCAT11_PageViewAnswersAnUnpublishedSlugLikeAMissingOne` (CAT11, P1).
  The anonymous `POST /page-views` used to answer `204` for a pending or
  private slug and `404` for one never filed, an enumeration oracle for
  exactly the slugs `GET /entries/{slug}` refuses to confirm. `RecordPageView`
  now checks the slug through `OpenEntry(ctx, slug, "")`, so the
  identical-404 invariant above holds on this door too and a view counts only
  against an entry an anonymous reader could open.
- `routes/sharelink_security_test.go` —
  `TestCAT12_ShareLinkIssuerIsNeverTakenFromTheRequestBody` (CAT12, P2). A
  share link's `created_by` used to be whatever the request body's `by` said.
  `issue_share_link` now binds `by` from `from: "caller"` —
  `mwanachama-backend-shared`'s `dispatch` gained that source and a
  `Deps.Caller`, and `routes.Mount` is where a host passes one. The module
  still holds no auth model: with no `Caller` supplied the issuer is `""`,
  never the request's.

Both are pinned at the mount as well, in `mwanachama-wakala-api`'s
`internal/api/http/catalog_issuer_security_test.go` — which is where the
end-to-end proof has to live, since `make self-test` mounts no catalog
(`buildCatalog` needs `CATALOG_DATABASE_URL` and `CATALOG_SPEC`, and this
family does not stand up a real Postgres to test).

Probed and holding on the same run, through `mwanachama-wakala-api`'s
router: a share key opens only the entry it was issued for (another private
entry answers the same 404), a key dies when its entry leaves `private` and
does not revive when it returns, and `PUT /entries/{slug}` ignores a body's
`visibility`, `approved_at`, `id` and `created_at` — a changed document
still lands `pending`. Those are pinned in wakala-api's
`catalog_storefront_security_test.go`.
