# QA

Test coverage and results for `mwanachama-backend-catalog`. The tests live
beside the code they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
make test-pg         # -tags=integration, with POSTGRES_URL set
```

As of 2026-09-23: 68 test functions, 119 counting subtests, all green —
`entry_test.go` (14), `spec/spec_test.go` (15), `suggestion_test.go` (12),
`run_test.go` (8), `routes/routes_test.go` (7), `private_test.go` (5),
`sharelink_test.go` (5), `domain_agnostic_test.go` (2). SQLite in memory is
the expected way to verify a change here — do not reach for a real Postgres.

## What is guarded, and how

- **The behaviours in [CLAUDE.md](../../CLAUDE.md)** each have a test, and
  the first two were mutation-checked rather than trusted: leaking a pending
  entry from the identical-404 refusal fails the test, and turning the stall
  counter's `>=` into `>` fails it.
- **The domain-agnostic rule** is enforced, not asserted:
  `domain_agnostic_test.go` parses `models/`, `gormstore/` and `spec/` and
  fails on any identifier carrying a domain word, with a second test over
  every stored enum value — because a stored value outlives a rename. It was
  mutation-checked by reintroducing `Entry.Sector`.
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

## What is not covered yet

`make test-pg` has nothing tagged `integration` to run. Three things exist
only on Postgres and are unguarded: the `jsonb` column type and the document
reads that go through it, the inner parentheses an expression index needs
(SQLite accepts a single pair, so `TestExpressionIndexesAreDoubleParenthesised`
asserts on the emitted string rather than on a database accepting it), and
identifier truncation at 63 bytes. Tracked as CAT2 on
[the board](../3.%20implementation/todo.md).

Two bugs so far were found by a real database rather than by these tests —
the missing inner parentheses, and a slug beginning with a digit being
dropped by an identifier pattern — which is the standing argument for CAT2.
`cmd/import` has also never been run against a live database; `cmd/seed`'s
SQL path has (623 agencies into `mwanachama-2`).
