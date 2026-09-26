# QA

Test coverage and results for `mwanachama-backend-catalog`. The tests live
beside the code they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
make test-pg         # -tags=integration, with POSTGRES_URL set
```

As of 2026-09-26: 98 test functions offline, all green — `spec/spec_test.go`
(15), `entry_test.go` (14), `suggestion_test.go` (12), `run_test.go` (8),
`routes/routes_test.go` (7), `private_test.go` (5), `sharelink_test.go` (5),
`store_test.go` (4), `blueprint_test.go` (3), `domain_agnostic_test.go` (3),
`routes/tools_test.go` (10), `spec/blueprint_test.go` (10),
`validate_test.go` (2), plus 7 more in `postgres_integration_test.go` that
need a database. SQLite in memory is the expected way to verify a change
here — reach for a real Postgres only for what the section below lists.

## What is guarded, and how

- **The behaviours in [CLAUDE.md](../../CLAUDE.md)** each have a test, and
  the first two were mutation-checked rather than trusted: leaking a pending
  entry from the identical-404 refusal fails the test, and turning the stall
  counter's `>=` into `>` fails it.
- **The domain-agnostic rule** is enforced, not asserted:
  `domain_agnostic_test.go` parses `models/`, `spec/` and the root package
  and fails on any identifier carrying a domain word, with a second test
  over every stored enum value — because a stored value outlives a rename.
  It was mutation-checked by reintroducing `Entry.Sector`. (`gormstore/`,
  the third parse target this once named, was deleted 2026-09-23 — CAT1.)
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
