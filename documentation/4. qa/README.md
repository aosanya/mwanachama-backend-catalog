# QA

Test coverage and results for `mwanachama-backend-catalog`. The tests live
beside the code they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
make test-pg         # -tags=integration, with POSTGRES_URL set
```

As of 2026-09-26 (`go test ./...` and `go test -tags=integration ./...`,
both green): 81 test functions offline — `entry_test.go` (14),
`suggestion_test.go` (12), `routes/tools_test.go` (10), `run_test.go` (8),
`examples_test.go` (8), `routes/routes_test.go` (7), `sharelink_test.go`
(5), `private_test.go` (5), `store_test.go` (4), `domain_agnostic_test.go`
(3), `blueprint_test.go` (3), `validate_test.go` (2) — plus 7 more in
`postgres_integration_test.go` that need a database, 88 in all.
`spec/spec_test.go` is gone from this count: S18 moved `spec/` itself to
`mwanachama-backend-shared` on 2026-09-24, and this repo now tests the
blueprint/example layer that sits on top of it (`blueprint_test.go`,
`examples_test.go`) instead. SQLite in memory is the expected way to verify
a change here — reach for a real Postgres only for what the section below
lists.

## What is guarded, and how

- **The behaviours in [CLAUDE.md](../../CLAUDE.md)** each have a test, and
  the first two were mutation-checked rather than trusted: leaking a pending
  entry from the identical-404 refusal fails the test, and turning the stall
  counter's `>=` into `>` fails it.
- **The domain-agnostic rule** is enforced, not asserted:
  `domain_agnostic_test.go` parses `models/`, `routes/` and the root package
  and fails on any identifier carrying a domain word, with a second test over
  every stored enum value and a third over the blueprint's own roles, fields
  and stored `values` — because a stored value outlives a rename. It was
  mutation-checked by reintroducing `Entry.Sector`. (`gormstore/` and the
  hand-written `spec/` it once parsed are both gone — see CLAUDE.md's "What
  is superseded" and "Objects are declared, not written".)
- **The spec engine's own refusals** — bad names, an underscore in a
  segment, a duplicate role, identifiers the database would merge past 63
  bytes, a required field carrying a default, unknown JSON keys — moved with
  `spec/` to `mwanachama-backend-shared` on 2026-09-24 (S18) and are now that
  repo's own `spec/spec_test.go` (`TestValidate_RefusesABadSpec`,
  `TestValidate_RefusesAnUnderscoreInASegment`,
  `TestValidate_RefusesADuplicateRole`,
  `TestValidate_RefusesIdentifiersTheDatabaseWouldMerge`,
  `TestRequiredFieldsHaveNoDefault`, `TestParse_RefusesUnknownKeys`, and
  `TestTwoDomainsCoexist`/`TestMigrateIsIdempotent` for two domains sharing
  one database). What stays in this repo is the domain-level equivalent
  against the two shipped catalog specs: `examples_test.go`'s
  `TestShippedExamplesCoexist`, `TestShippedExampleMigratesIdempotently`,
  `TestShippedIndexesComeFromTheDomain`, `TestShippedCompositePrimaryKeys`,
  `TestShippedExamplesHaveNoDefaultedRequiredColumn`,
  `TestShippedExamplesAreDescribed`, and
  `TestShippedExpressionIndexesAreDoubleParenthesised` (a copy of the engine
  test's assertion, run against `agency.catalog.json`/`library.catalog.json`
  specifically), plus `blueprint_test.go`'s `TestEveryExampleFitsTheTypes`
  and `TestExamplesDeclareNoFieldsOfTheirOwn` (CAT5's "a domain names
  objects; it does not re-declare them" rule).
- **`routes/routes_test.go`** was the acceptance test the W14 dispatcher had
  to pass **unchanged** when it replaced the twelve hand-written handlers
  (CAT4/W14, 2026-09-23) — `routes.go` is now 74 lines of adapter over
  `mwanachama-backend-shared/dispatch`, and this file still passes against it
  unmodified, real `http.Client` through a real `http.ServeMux`.
  `routes/tools_test.go` covers `routes.BuildTools`, the same
  `catalog.operations.json`-declared operations built as callable tools
  instead of HTTP handlers: every gated operation becomes one
  (`TestEveryGatedOperationBecomesATool`), a reader operation does not
  (`TestTheReadersOperationsAreNotTools`), each carries a schema
  (`TestEveryToolCarriesProseAndASchema`,
  `TestEveryDescribedFieldIsOneItsTypeCarries`), and a tool refuses an
  argument its operation never declared
  (`TestAToolRefusesAnArgumentTheOperationDoesNotDeclare`).

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
