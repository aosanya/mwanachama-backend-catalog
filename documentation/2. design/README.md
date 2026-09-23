# Design

The org-wide shape this module is the first to take is written up one level
up, in
[architecture-spec-driven-modules.md](../../../developer/documentation/2.%20design/architecture-spec-driven-modules.md).
This page records only what is true of this repo today.

## Objects are declared, not written

`spec/` holds the spec types, loader, validation and migrator. A spec is one
domain's declared shape — `module`, `domain`, `instance` and a list of
objects — and `spec.Load` refuses a bad one at startup, the only moment at
which refusing it is cheap. `spec.Migrate` creates a table per declared
object and its indexes, idempotently; `spec.DDL(dialect)` prints the same
statements for review, which is what `cmd/ddl` does.

An object is named for the domain's own noun — "agency", "title",
"mining_run" — and its table is that noun's plural. What connects a domain's
noun to a module rule is a separate `role`: the visibility check asks for
whichever object fills `"entry"` and gets agencies in one domain, titles in
another (`Spec.ByRole`). An object with no role is the domain's own, and no
rule in the module touches it. Two objects claiming one role is refused, and
`Spec.RequireRoles` lets a module state what it needs so an unfilled role
stops the process starting rather than failing a request.

Field types are `string`, `text`, `int`, `bool`, `json`, `timestamp` and
`enum`. An index is either over columns or over a document path, which is how
a domain makes its own vocabulary fast — `_template.sector`,
`identifiers.isbn` — without this module learning the word.

`spec/examples/agency.catalog.json` and `library.catalog.json` are the same
module under two domains; `TestTwoDomainsCoexist` migrates both into one
database.

## The module declares its own objects, once (CAT5)

The nine objects are the module's, not a domain's. They were declared once per
domain until 2026-09-23, and the two example specs had drifted into 62 field
declarations each of which 60 were identical bar the description — a second
domain was copying the module's whole shape to change two defaults and nine
names. `models/` was a third copy in Go, cross-checked only against whichever
spec happened to be loaded, so a field renamed under one domain left the other
wrong with nothing failing until someone loaded it.

`catalog.blueprint.json` at the repo root is now the single declaration. It is
embedded into the root package (`//go:embed`) and reached through
`catalog.Blueprint()`, `catalog.LoadSpec(path)` and `catalog.ParseSpec(raw)`.
A consumer names a spec file and nothing else; the blueprint travels with the
module.

What each side owns:

| The module's blueprint | A domain's spec |
| --- | --- |
| the roles that exist | which object fills each role, and what it is called |
| every field: name, type, description, `required`/`unique`/`immutable`/`primary`, `values`, `matches` | `instance`, and the table each object lands in |
| the indexes every domain needs | its own indexes, including document-path ones over its own vocabulary |
| — | a **default**, and nothing else, on a field it fills a role with |
| — | objects of its own, which declare their own fields and take no role |

A domain that sets anything but a default on a declared field is refused, by
name, at load. So is one that claims a role the module does not declare, or
adds a field to one it does — a domain that needs a field of its own declares
an object of its own, the way `library.catalog.json` declares `copy`.

`spec.Blueprint` does the resolving: `ParseBlueprint`/`LoadBlueprint` read it,
`Blueprint.Load`/`Blueprint.Parse` read a domain spec through it, and
`Blueprint.Apply` is the merge. A spec loaded with plain `spec.Load` still
works, and one that relies on the blueprint says so in the error rather than
reporting an object with no fields.

Three guards, each mutation-checked:

- `TestEveryExampleFitsTheTypes` loads **every** spec in `spec/examples/`,
  migrates it and builds a manager over it, so drift under a domain nobody
  loaded fails the build rather than waiting. Renaming one blueprint field
  fails it under both domains at once.
- `TestVocabularyMatchesTheBlueprint` holds `models/vocabulary.go`'s constants
  and the blueprint's declared `values` to exactly each other, in both
  directions. A stored value outlives a rename, so these may not drift apart.
- `TestExamplesDeclareNoFieldsOfTheirOwn` fails the moment a domain spec
  starts restating a type, a description or a value set.

`TestNoDomainWordsInTheBlueprintsRolesFieldsAndStoredValues` extends the
domain-word rule to the blueprint, since the module's declaration is data now.
Descriptions are exempt, the way comments are.

**What the Go types are for.** `models/` is the typed thing a caller holds,
and it is no longer a free copy: `newStore` refuses at construction when a
declared column has no field or a field has no column, and the test above runs
that check against every shipped domain. Generating the structs from the
blueprint would remove the last restatement, and was deliberately not done —
it would change public field types (`int` is `bigint`, so every count would
become `int64`) on a module other repos already import, to buy little now that
the agreement is checked rather than assumed.

## Names reach the database as SQL text

A table name cannot be a bound parameter, so the spec refuses rather than
escapes:

- `spec.NamePattern` for object, table, field and index names.
- `spec.SegmentPattern` for `instance` and `module` — stricter by exactly the
  underscore, because `<instance>_<module>_<table>` cannot be read back apart
  if the first two segments may contain one.
- `spec.DocPathPattern` for a document path, which reaches SQL inside
  `doc #>> '{a,b}'` / `json_extract(doc, '$.a.b')`. Anything else is
  `ErrInvalidDocPath`. `models.DocFilter.Valid` defers to it rather than
  keeping a second copy, and the module's other named patterns — `slug`,
  `json_object` — live in `patterns.go` beside the table that names them.
- `spec.MaxIdentifier = 63`, because Postgres truncates an identifier at 63
  bytes without complaining — two objects differing only past the cut would
  silently share one table, and an index name (`<table>_<index>_idx`) is
  always longer than the table it derives from.

Never relax any of these to "escape it instead".

**A required field must not carry a default.** The default is exactly what
would let an omitted value pass unnoticed; `TestRequiredFieldsHaveNoDefault`
asserts no column is ever both `not null` and defaulted.

## Table naming

`Spec.TableFor` is `<instance>_<module>_<table>` — `agency_catalog_agencies`
— and `Spec.IndexFor` is that plus `_<index>_idx`, since Postgres keeps
indexes in the same per-schema namespace as tables. The module segment is
what lets two modules share one database without arranging not to collide:
GORM's `AutoMigrate` adopts a table that already exists and adds the missing
columns without error, and `create table if not exists` is a no-op against
one, so a collision is silent either way. The wider decision, spanning
wakala-api's registry too, is W16 on the website's board.

## The document column

`json` is `jsonb` on Postgres and `text` everywhere else — a column declared
`jsonb` on SQLite lands in NUMERIC affinity, which is not what anyone means.
Filtering and counting inside the document happen in the database:
`doc #>> '{a,b}'` on Postgres, `json_extract` on SQLite, which has JSON1. An
expression index needs its own inner parentheses on Postgres — `on t ((expr))`
— and SQLite accepts a single pair, which is why
`TestExpressionIndexesAreDoubleParenthesised` exists.

## What the spec does not declare

The logic a spec cannot state stays Go, in the root package: the identical-404
enumeration refusal, the stall counter, the write-once share key,
`PutPrivate`'s merge-don't-erase, the document-free listing. The spec may only
narrow what the manager refuses, never widen it.

## Packages, and which are superseded

| Package | State |
| --- | --- |
| `catalog.blueprint.json` | The module's own objects, declared once — CAT5. |
| `spec/` | The spec types, the loader, validation, the blueprint merge and DDL. |
| root (`entry.go`, `suggestion.go`, `sharelink.go`, `private.go`, `run.go`, `catalog_manager.go`, `errors.go`, `blueprint.go`, `patterns.go`) | The rules, the named patterns, and the public `CatalogManager`. |
| `models/` | The typed things a caller holds, plus what a spec cannot state. Shrunk by CAT3, stripped of its restated prose by CAT5. |
| `gormstore/` | Deleted — CAT1. `spec.Migrate` and `store.go`'s codec replaced it. |
| `routes/` | Twelve hand-written handlers, rejected by the owner and superseded by the dispatcher (W14). **Do not copy.** `routes_test.go` survives as the dispatcher's acceptance test — CAT4. |
| `cmd/ddl`, `cmd/import`, `cmd/seed` | Print a spec as SQL; import a domain's folders into a live database; emit the same folders as upsert SQL for a deployment whose schema goes through its own migration tooling. |

## The nine objects

`Entry` + its document, `EntryPrivate`, `ShareLink`, `Suggestion`,
`SuggestionComment`, `SuggestionMatch`, `MiningRun`, `Setting`, `PageView`.
What each is under two domains is tabulated in [doc.go](../../doc.go) and
[README.md](../../README.md).
