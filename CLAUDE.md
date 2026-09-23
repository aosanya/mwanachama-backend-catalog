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

`domain_agnostic_test.go` enforces this. It parses `models/`, `spec/` and the
root package, and fails on any identifier carrying a domain word, with a
second test over every stored enum value. Domain words stay allowed in comments,
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

**A table is `<instance>_<module>_<object>`** — `agency_catalog_agencies`.
The module segment is not decoration: without it a catalog instance named
`agency` and an agency instance of the same name both want `agency_agencies`,
and neither module notices, because `AutoMigrate` adopts a table that already
exists and `create table if not exists` is a no-op against one. A collision
here is silent data mixing, not an error, which is why `spec.Validate` refuses
it rather than leaving it to the database:

- `Instance` and `Module` match `SegmentPattern` — `NamePattern` minus the
  underscore, so the three segments can be read back apart.
- Every emitted name is measured against `MaxIdentifier` (63). Postgres
  truncates past it **without complaining**, so two names agreeing that far
  are one relation — with a long agency slug, `…_catalog_entries` and
  `…_catalog_entry_privates` are the same table.
- Duplicates are detected on the physical name, and indexes are checked in
  the same namespace as tables, because Postgres keeps them there — and
  `<table>_<index>_idx` overflows before its own table does.

**A column is found by field name, never by json tag.** `store.go`'s
`columnName` turns `SubmittedBy` into `submitted_by` and `RunID` into
`run_id`. The tag is a presentation choice and gets this wrong where it
hurts: `ShareLink.KeyHash` is `json:"-"` so a stored hash never reaches an
API response, and a tag-reading codec would quietly stop storing it — every
share key would then fail to match. Mutation-checked in `store_test.go`; the
tag-reading variant breaks five share-key tests.

**Every declared column is written on every write.** A map missing a key
means "leave it alone" to an update, so omitting empty values would make
clearing a field impossible — an Entry would keep the `approved_at` that its
document changing is supposed to drop.

**The spec and the types are checked against each other when the manager is
built.** A declared column with no field to hold it, or a field with no
column to land in, fails in `NewCatalogManager` rather than dropping a value
on every write.

## Rules are declared too, where they can be

`validate.go` reads `required`, an enum's `values` and `matches` off the
spec and applies them on the way in. So the permitted states are the
domain's to declare: a spec that adds a fourth visibility gets one with no
Go change, which `TestVocabularyComesFromTheSpec` demonstrates both ways.
`models/vocabulary.go` keeps only the constants Go compares against — the
membership maps and their `IsX` functions are gone.

`matches` names a pattern; this module says what the name means, in
`patterns` (`slug`, `json_object`). A spec naming one nobody supplies is an
error, not a rule that quietly never runs.

**What stays in Go is what a spec cannot say**, and each piece lives with
the type it is about: `PageView.SlugFitsPage` (one field's presence depends
on another's value), and the non-negative checks on `Rank`, `Attempts` and
`TemplatesFiled`.

`Check(spec, role, value)` is the same validation without a database, for a
bulk import validating what it has read before it opens a connection.

**One pattern, one expression builder.** `models.DocFilter.Valid` uses
`spec.DocPathPattern`, and `docPathExpr` defers to `spec.DocPathExpr` — the
same function the migrator builds a document index with. Both were copies
once. The second matters for more than tidiness: an expression index applies
only to a query whose expression matches it exactly, so a copy that drifted
by a space would leave every document filter doing a full scan with nothing
reporting it.

## What is superseded

- **`gormstore/`** — **deleted 2026-09-23 (CAT1)**. Its nine row structs and
  `Migrate` are what `spec.Migrate` and `store.go`'s codec replace; root
  `tables.go` and its `TableNames`/`DefaultTableNames` re-export went with
  it. `NewCatalogManager(db, *spec.Spec)` is the constructor now, and the
  package's own tests run against the shipped
  `spec/examples/agency.catalog.json` rather than a test-only table set.
- **`routes/`** — twelve hand-written handlers, rejected by the owner and
  superseded by the spec-driven dispatcher (W14). **Do not copy this
  package.** `routes/routes_test.go` survives as the acceptance test the
  dispatcher must pass unchanged.
- **`models/`** — shrunk 2026-09-23 (CAT3): the six membership maps, their
  `IsX` functions, four `Validate` methods and a duplicate `DocPathPattern`
  are gone. What remains is the types themselves, the constants Go compares
  against, and the rules a spec cannot state.

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
