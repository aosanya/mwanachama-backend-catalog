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

## Names reach the database as SQL text

A table name cannot be a bound parameter, so the spec refuses rather than
escapes:

- `spec.NamePattern` for object, table, field and index names.
- `spec.SegmentPattern` for `instance` and `module` — stricter by exactly the
  underscore, because `<instance>_<module>_<table>` cannot be read back apart
  if the first two segments may contain one.
- `spec.DocPathPattern` (and `models.DocPathPattern` on the manager side) for
  a document path, which reaches SQL inside `doc #>> '{a,b}'` /
  `json_extract(doc, '$.a.b')`. Anything else is `ErrInvalidDocPath`.
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
| `spec/` | The source of truth for objects, indexes and DDL. |
| root (`entry.go`, `suggestion.go`, `sharelink.go`, `private.go`, `run.go`, `catalog_manager.go`, `errors.go`) | The rules, and the public `CatalogManager`. |
| `models/` | Domain types. Shrinks to what the rules genuinely need — CAT3. |
| `gormstore/` | Nine row structs, converters and `Migrate` — what `spec.Migrate` replaces. Superseded, not yet retired — CAT1. |
| `routes/` | Twelve hand-written handlers, rejected by the owner and superseded by the dispatcher (W14). **Do not copy.** `routes_test.go` survives as the dispatcher's acceptance test — CAT4. |
| `cmd/ddl`, `cmd/import`, `cmd/seed` | Print a spec as SQL; import a domain's folders into a live database; emit the same folders as upsert SQL for a deployment whose schema goes through its own migration tooling. |

## The nine objects

`Entry` + its document, `EntryPrivate`, `ShareLink`, `Suggestion`,
`SuggestionComment`, `SuggestionMatch`, `MiningRun`, `Setting`, `PageView`.
What each is under two domains is tabulated in [doc.go](../../doc.go) and
[README.md](../../README.md).
