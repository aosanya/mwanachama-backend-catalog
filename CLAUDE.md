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

`domain_agnostic_test.go` enforces this. It parses `models/`, `routes/` and the
root package, and fails on any identifier carrying a domain word, with a
second test over every stored enum value. Domain words stay allowed in comments,
where they illustrate what a neutral name stands for. When adding a field,
ask whether a librarian would recognise it.

## Objects are declared, not written

The tables come from a JSON spec, not from Go structs — see
`developer/documentation/2. design/architecture-spec-driven-modules.md`.

- `catalog.blueprint.json` — **the module's nine objects, declared once**.
  Embedded into the root package and reached through `catalog.Blueprint()`,
  `catalog.LoadSpec(path)` and `catalog.ParseSpec(raw)`. Load a domain spec
  through those, never through `spec.Load`, or its roled objects arrive
  with no fields.
- `mwanachama-backend-shared/spec` — the spec types, loader, validation,
  blueprint merge and migrator. `spec.Load` refuses a bad spec at startup;
  `spec.Migrate` creates a table per declared object and its indexes,
  idempotently. It lived in this repo's own `spec/` until 2026-09-24, when it
  moved to shared (S18) so the other backends build on the same engine; what
  remains under `spec/` here is `examples/`.
- `spec/examples/agency.catalog.json` and `library.catalog.json` — the same
  module under two domains. `TestTwoDomainsCoexist` migrates both into one
  database.
- A domain's own vocabulary becomes an index by path
  (`_template.sector`, `identifiers.isbn`) without this module learning the
  word.

**A domain names objects; it does not re-declare them (CAT5).** Its spec
supplies `instance`, the name and table each role lands in, its own indexes,
and a **default** on a declared field — nothing else. Setting a type, a
description or a value set on a field the module declares is refused by name
at load, as is claiming a role the module does not declare or adding a field
to one it does. A domain that needs a field of its own declares an object of
its own, with no role, the way `library.catalog.json` declares `copy`. Before
this, both example specs carried all 62 field declarations, 60 of them
identical bar the description.

**Adding a field means editing the blueprint and the Go type together**, and
`TestEveryExampleFitsTheTypes` runs that agreement against *every* spec in
`spec/examples/`, not just the one a test happened to load — the drift it
closes was invisible under the domain nobody loaded. Never add a field to a
domain spec to get it into the module.

**Identifiers and paths reach the database as SQL text** — a table name
cannot be a bound parameter — so `spec.NamePattern` and `spec.DocPathPattern`
restrict them to strict alphabets and anything else is refused outright
rather than escaped. Never relax this to "escape it instead".

**A required field must not carry a default.** The default is exactly what
would let an omitted value pass unnoticed; `TestRequiredFieldsHaveNoDefault`
asserts no column is ever both `not null` and defaulted.

**A table is `<instance>_hashOf(<mount>)_hashOf(<module>_<object>)`** — only
the instance stays readable, e.g. `agy1f2e3d4c_0d6e4079_7fc95a982afe`. The
middle segment groups one mount's tables together; the last identifies the
table, and reads the same in every instance. `spec.Migrate` records both the
physical and the raw name in the per-instance `<instance>_spec_table_names`
registry. Hashing keeps every identifier a constant 33 bytes (57 for an index)
against Postgres's 63-byte ceiling, which the readable form had already
reached. Assert on `RawNameFor` in tests, never on a physical name literal, and
exclude `%_spec_table_names` from anything that counts tables. See
[declared-domains.md](../mwanachama-backend-shared/documentation/2.%20design/declared-domains.md).


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

**A domain spec may attach another module to one of its roles** (shared's
S62, 2026-10-12): `"attach": [{"module": "ratings", "items": "catalog.entry"}]`
makes every entry rateable, with ratings mounted beside the catalog by the
host. This module carries no Go that knows an attachment exists — the
engine validates the block and the host acts on it — so nothing here names
ratings, and `domain_agnostic_test.go` is unaffected. See
[declared-domains.md](../mwanachama-backend-shared/documentation/2.%20design/declared-domains.md#a-domain-attaches-another-module).

## Rules are declared too, where they can be

`validate.go` reads `required`, an enum's `values` and `matches` off the
spec and applies them on the way in. So the permitted states are the
domain's to declare: a spec that adds a fourth visibility gets one with no
Go change, which `TestVocabularyComesFromTheSpec` demonstrates both ways.
`models/vocabulary.go` keeps only the constants Go compares against — the
membership maps and their `IsX` functions are gone.

`matches` names a pattern; this module says what the name means, in
`patterns.go` (`slug`, `json_object`). A spec naming one nobody supplies is
an error, not a rule that quietly never runs. The patterns live there and not
beside a type, because a pattern reached by name from a declaration is the
module's, not one struct's.

**What stays in Go is what a spec cannot say**, and each piece lives with
the type it is about: `PageView.SlugFitsPage` (one field's presence depends
on another's value), and the non-negative checks on `Rank`, `Attempts` and
`TemplatesFiled`. `SlugFitsPage` checks presence only — the slug's *shape* is
`matches: "slug"` on the blueprint's `page_view.slug`, applied like any other
declared rule.

`TestVocabularyMatchesTheBlueprint` holds `models/vocabulary.go` and the
blueprint's declared `values` to exactly each other, in both directions. The
constants were promised to agree with the spec and nothing checked it.

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
- **`routes/`** — the twelve hand-written handlers are **deleted 2026-09-23
  (CAT4/W14)**. The route table is declared in `catalog.operations.json` and
  built by `mwanachama-backend-shared/dispatch`; `routes.go` is 74 lines of
  adapter and `routes_test.go` passed it unchanged, which was the acceptance.
  The four constructors are gone on purpose: the gate boundary they encoded
  is now `AnonymousActions`, an allowlist of action ids, so a mount names
  actions rather than this module's Go identifiers and anything unnamed
  arrives gated. See [documentation/2. design/routes.md](documentation/2.%20design/routes.md).
- **`models/`** — shrunk 2026-09-23 (CAT3): the six membership maps, their
  `IsX` functions, four `Validate` methods and a duplicate `DocPathPattern`
  are gone. Shrunk again the same day (CAT5): `SlugPattern` and
  `IsJSONObject` moved to `patterns.go`, and the per-field doc comments went
  — they were a hand-rewritten copy of the blueprint's `description`, which
  is where that prose lives now. What remains is the types themselves, the
  constants Go compares against, and the rules a spec cannot state.
- **A second copy of the object set in a domain spec** — gone (CAT5). The
  field list is the module's; see "Objects are declared, not written".

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
- Re-sending an **unchanged** document, name and summary leaves its review
  alone, and the write leaves `visibility` and `approved_at` out of the row so
  a reviewer's concurrent decision survives (CAT18, CAT35); a changed one
  returns to pending and retires every live key (CAT39, CAT42). `UpdateEntry`
  follows the same rule and ignores any visibility the caller sends (CAT36).
  **"Unchanged" is semantic, not byte-wise** — `jsonb`
  sorts keys and strips whitespace, so the document read back is rarely the
  one sent, and the byte compare this started as silently un-published every
  re-imported agency on Postgres while passing on SQLite. `sameDocument`
  compares canonical JSON; only `postgres_integration_test.go` catches a
  regression here.
- An Entry page view counts only against an entry an anonymous reader could
  open, and every other slug answers exactly as a missing one (CAT11).

## Conventions

- `go test ./...` (sqlite via `glebarez/sqlite`) is the expected way to
  verify a change here — do not reach for a real Postgres. The `jsonb`
  behaviours SQLite cannot stand in for live in `postgres_integration_test.go`
  (`//go:build integration`, gated on `POSTGRES_URL`, run by `make test-pg`),
  which is **not** part of `go test ./...`. The tag is `integration`, not the
  sibling repos' `postgres`, because the Makefile target came first. See
  [[feedback_use_memory_backend_for_tests]].
- No Go file over 300 lines; split by responsibility.
- There are no hand-written route builders left to name: an address is an
  entry in `catalog.operations.json`, and adding one is an edit to that file
  rather than a new Go function. [[feedback_routes_name_match_model]] still
  governs any repo that has not adopted the dispatcher.
- This module has **no auth model**, matching
  `mwanachama-backend-assetmanager`: a route still needs a capability gate
  wrapped around it by whatever mounts it. The role→action grant table is
  `mwanachama-backend-auth`'s (DEV-1700).

## Code comments

Write code with no comments. Not one-liners above a function, not section
banners, not doc comments on exported symbols, not "why" notes next to a
tricky line. A name, a type, or a smaller function carries it instead.

Anything that genuinely needs explaining goes in this repo's `documentation/`
folder, under the phase it belongs to (`1. requirements`, `2. design`,
`3. implementation`, `4. qa`) — never inline.

**Why:** inline prose drifts out of sync with the code, duplicates what
`documentation/` already owns, and buries the explanation where nobody
looking for it will search.

**How to apply:**

- New code ships without comments. If a line seems to need one, rename or
  split until it doesn't.
- Touching code that already has comments: strip the ones in the code you are
  changing. Do not sweep untouched files unless asked.
- If the reasoning matters, add or update the matching `documentation/` page
  in the same change and leave nothing behind in the source.
- Machine-read directives are not comments and stay: build tags, `//go:embed`,
  `//go:generate`, linter pragmas (`//nolint`, `// eslint-disable-next-line`,
  `// ignore:`), license headers, codegen "do not edit" banners, and generated
  files as a whole.
- Commit messages, PR descriptions, and test names carry the narration that
  used to go in comments.

This rule is repeated verbatim in every mwanachama repo's `CLAUDE.md` so that
it reaches sessions that do not load this machine's user-level config —
scheduled cloud routines, other machines, and other agent harnesses.
