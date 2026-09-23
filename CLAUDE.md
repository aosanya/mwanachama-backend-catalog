# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-catalog

A catalog of documents with a review state: things are proposed, built,
reviewed, published or shared under a key, and counted. Domain logic AND
storage both live in this package, imported directly by whatever mounts it —
no separate service, no gRPC, no proto — the same shape
`mwanachama-backend-actor`, `mwanachama-backend-assetmanager` and
`mwanachama-backend-forms` already took. Module path
`github.com/aosanya/mwanachama-backend-catalog`.

It replaces 11 hand-written tables and ~33 `security definer` functions that
lived only in `mwanachama-website/supabase/migrations/`. Tracked as W13 and
W14 on `mwanachama-website`'s board.

## The rule that matters most: this module names no domain

**A word that means something in one domain and nothing in another does not
belong in this module.** "Sector" fails it — a library has genres. "Entry"
passes it.

It was written for an agency storefront and must run a library or a course
catalog without a line changing. Three things failed that test on 2026-09-23
and were moved out; they are the pattern to check every addition against:

| Was | Why it failed | Where it went |
| --- | ------------- | ------------- |
| `Entry.Sector`, `Entry.Archetype` — columns | agency words | inside the document, filtered by JSON path |
| `Suggestion.Sector` — a column | same | inside `Proposal` |
| `PageKind` values `"home"` / `"agency"` | the second is an agency word | `"index"` / `"entry"` |

The third is the worst kind: **a stored enum value outlives a rename.**

`domain_agnostic_test.go` enforces this. It parses `models/`, `gormstore/`
and `spec/` and fails on any identifier carrying a domain word, with a second
test over every stored enum value. Domain words stay allowed in comments,
where they illustrate what a neutral name stands for. When adding a field,
ask whether a librarian would recognise it.

## Objects are declared, not written

The tables come from a JSON spec, not from Go structs — see
`developer/documentation/2. design/architecture-spec-driven-modules.md`.

- `spec/` — the spec types, loader, validation and migrator. `spec.Load`
  refuses a bad spec at startup; `spec.Migrate` creates a table per declared
  object and its indexes, idempotently.
- `spec/examples/agency.catalog.json` and `library.catalog.json` — the same
  module under two domains. `TestTwoDomainsCoexist` migrates both into one
  database.
- A domain's own vocabulary becomes an index by path
  (`_template.sector`, `identifiers.isbn`) without this module learning the
  word.

**Identifiers and paths reach the database as SQL text** — a table name
cannot be a bound parameter — so `spec.NamePattern` and `spec.DocPathPattern`
restrict them to strict alphabets and anything else is refused outright
rather than escaped. Never relax this to "escape it instead".

**A required field must not carry a default.** The default is exactly what
would let an omitted value pass unnoticed; `TestRequiredFieldsHaveNoDefault`
asserts no column is ever both `not null` and defaulted.

## What is superseded

- **`gormstore/`** — its row structs and `Migrate` are what `spec.Migrate`
  replaces. `JSONDoc`'s dialect split already moved to `spec/migrate.go`.
- **`routes/`** — twelve hand-written handlers, rejected by the owner and
  superseded by the spec-driven dispatcher (W14). **Do not copy this
  package.** `routes/routes_test.go` survives as the acceptance test the
  dispatcher must pass unchanged.
- **`models/`** — shrinks to whatever the rules genuinely need. Open.

What stays Go is the logic a spec cannot declare: the enumeration refusal,
the stall counter, the write-once key, the merge-don't-erase in
`PutPrivate`.

## Behaviours to preserve

Each has a test, and the first two were mutation-checked rather than trusted:

- A missing slug, a pending Entry, and a private Entry opened with a wrong or
  absent key are all the same `ErrEntryNotFound`. Splitting them lets anyone
  enumerate the unpublished catalog by probing slugs.
- The tenth failed build stalls a suggestion; approving it again clears the
  count. `>=` not `>`.
- A share key is stored only as its SHA-256; the raw key is returned once.
- Leaving `private` retires every live key.
- A listing never carries `Doc`.
- Re-sending an **unchanged** document leaves its review alone; a changed one
  returns to pending.
- An Entry page view counts only against a slug that exists.

## Conventions

- `go test ./...` (sqlite via `glebarez/sqlite`) is the expected way to
  verify a change here — do not reach for a real Postgres. The `jsonb`
  behaviours SQLite cannot stand in for belong in a Postgres integration
  test. See [[feedback_use_memory_backend_for_tests]].
- No Go file over 300 lines; split by responsibility.
- Route builder functions are named `<ModelType>Routes` — see
  [[feedback_routes_name_match_model]] — until `routes/` is deleted.
- This module has **no auth model**, matching
  `mwanachama-backend-assetmanager`: a route still needs a capability gate
  wrapped around it by whatever mounts it. The role→action grant table is
  `mwanachama-backend-auth`'s (DEV-1700).
