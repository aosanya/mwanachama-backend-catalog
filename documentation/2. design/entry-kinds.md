# Entry kinds (CAT59)

A mount may say which kinds of entry it holds and which fields each kind's
document carries. It is the `kinds` setting, declared in
`catalog.blueprint.json` through shared's S64 settings engine, defaulting to
no kinds — so a mount that sets nothing behaves exactly as before.

## The setting

```json
[
  {"name": "place", "fields": [
    {"name": "address", "type": "string", "required": true},
    {"name": "seats", "type": "number"}
  ]}
]
```

A field's type is one of `string`, `number`, `boolean`, `object`, `array`,
the JSON types. Names are unique per setting and per kind, and no field may
be called `kind`. A value breaking any of those is `models.ErrKindsMalformed`,
answered as a server error on save rather than silently skipping the check.

The value resolves through the stored layers on every save
(`spec.ResolveStored`): the module default, then the definition's value
(`spec.PutDefinitionSettings`), then an override (`spec.StoreOverride`).

## Where an entry names its kind

Entries carry no kind column, and adding one would put a word into the
blueprint for every mount. The kind lives in the **document**, at its
top-level `kind` key, beside the fields it describes. That is the smallest
neutral choice: the document is already the domain's, and a domain wanting
to filter on kind declares a document index on `kind` in its own spec.

## The check

On `CreateEntry`, `UpsertEntry` and `UpdateEntry`, after the existing entry
checks:

- the document has no `kind` key → not checked, and the setting is not read;
- the mount declares no kinds → not checked;
- `kind` is not text, names no declared kind, misses a required field, or
  carries a declared field of the wrong JSON type → `ErrEntryKind`, 400.

Fields the kind does not declare are allowed; a kind lists what must be
there and what shape it has, not everything a document may say.

## Seed rows

A mount may also say which entries it starts with. It is the `seeds`
setting, declared beside `kinds` through the same S64 engine and resolved
through the same stored layers, defaulting to none.

```json
[
  {"slug": "town-hall", "name": "Town hall", "summary": "Main building",
   "doc": {"kind": "place", "address": "1 Lane"}}
]
```

A seed is a slug, a name, an optional summary and an optional document
object, and nothing else: a seed naming any other field (a visibility, an
approval) is `models.ErrSeedsMalformed`, as is a missing slug or name, a
repeated slug, or a document that is not an object.

`CatalogManager.SeedEntries` applies them and answers the slugs it created.
Each seed goes through `CreateEntry`, so it lands **pending** like every new
entry and its document is checked against its kind. A seed whose slug an
entry already holds is skipped, so applying twice creates nothing the second
time and an existing entry is never overwritten. A soft-deleted entry still
holds its slug (CAT37), so a deleted seed does not come back. A seed that
fails its check stops the run with the error; the seeds before it stay
created, and the next run picks up where it left off.

The module does not decide when seeds apply. The host calls `SeedEntries`
when it means a mount to start with them.

## The first real case (CAT49)

The municipality's register mount is the first mount to declare kinds
(member app decisions 228 and 291). Its value is a definition setting, held
in the template's module config
(`developer/documentation/4. qa/agencies/municipality/modules/register_catalog.json`,
under `settings.kinds`) and written to the definition layer with
`spec.PutDefinitionSettings`. Nothing in this module names it.

It declares four kinds, `person`, `household`, `business_trader` and
`property`. Every kind carries `reference`, `ward`, `source` and `consent`
(required) and `language`, `reach` and `accessibility`; a household adds its
required `members` array and an optional `property`; a business or trader
adds an optional `property`; a property adds its required `address` and an
optional `use`. `property` holds the referenced property's `reference`.

A record is queried by field with the existing document filter
(`ListFilter.Docs`, `{path, value}`). Because the check reads the document's
top-level keys, the register's spec indexes the same top-level paths
(`reference` unique, `ward`, `consent`, `reach`, `source`, `kind`,
`property`); they had pointed at `_person.*` and `_record.*`, which no
checked document carries.

Checked on 2026-10-24 by loading that spec, writing the setting and saving
one record of each kind, three refused ones, and four filtered listings, on
SQLite. One thing is not done here:

- **The host does not yet write a template's `settings` to the definition
  layer.** No importer reads module configs (the template's own `status`
  says so); until one does, a running mount holds no kinds and checks
  nothing.

## A mount declares only the roles it uses (decision 443)

`entry`, `entry_private` and `share_link` are the roles every catalog
declares. Every other role is optional: `suggestion`, `suggestion_comment`,
`suggestion_match`, `run`, `setting` and `page_view` join `record`,
`endorsement`, `raised_item` and the rest, so `NewCatalogManager` builds over
a spec that leaves any of them out and checks a carrier only for the roles
the spec declares. The register's set — entry, entry_private, share_link,
setting, suggestion and suggestion_match — builds a manager as it stands.

A method that needs an undeclared role answers `ErrRoleNotDeclared`, naming
the role, mapped to 404 on the wire, before it reaches a table. Three
choices are worth knowing:

- **`GetSetting` with no `setting` role answers `ErrRoleNotDeclared` and
  `ErrSettingNotFound` together**, so every caller that falls back to a
  default on a missing setting (the answer clock, one record per holder)
  keeps doing so on a mount that keeps no settings. `PutSetting` refuses.
- **A failed build attempt still counts with no `suggestion_comment`
  role.** `RecordSuggestionAttempt` moves the counter and the stall as
  before and skips the comment it would have logged; commenting by hand
  and listing comments refuse.
- **Attaching, fulfilling and listing matches need `suggestion_match`**, and
  `ListEntryMatches` needs only that role.

`declared_roles_test.go` builds a manager over the register's set and over
one with neither settings nor suggestions, runs the entry operations through
both, and holds each refusal to `ErrRoleNotDeclared`.

## The host writes a template's settings through `PutDefinition`

A template's `settings` (its `kinds`, its `seeds`) reach the definition layer
only when the host writes them. `PutDefinition(db, spec, values)` is that one
call: it parses `kinds` with the same rules the write check reads and refuses
a malformed set with `ErrKindsMalformed` before anything is stored, then
replaces the definition layer through `spec.PutDefinitionSettings`. A mount
whose host has not called it checks nothing, because the module default for
`kinds` is empty. `definition_test.go` holds both halves.

## Left open

- **When seeds apply** — on provisioning only, on every boot, or on a
  publish of the definition — is the host's call and not decided here; nor
  whether a seed should arrive public rather than pending, or whether a
  deleted seed should return. Each is an owner question.
- **A stale override after a republish** — an override written against an
  older definition of `kinds` that no longer matches it — is an open owner
  question and is not decided: the override simply wins, as S64 resolves it.
