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

## Left open

- **When seeds apply** — on provisioning only, on every boot, or on a
  publish of the definition — is the host's call and not decided here; nor
  whether a seed should arrive public rather than pending, or whether a
  deleted seed should return. Each is an owner question.
- **A stale override after a republish** — an override written against an
  older definition of `kinds` that no longer matches it — is an open owner
  question and is not decided: the override simply wins, as S64 resolves it.
