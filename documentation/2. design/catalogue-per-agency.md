# A catalogue per agency per mount

CAT60, from the member app's decision 285 (owner, 2026-10-22): each agency
has catalogues of its own, as many as it needs (Tours and Guides, a term's
classes, tenders), in its own tables like its forms and register. Wakala's own
shop stays the shared one wakala-api already mounts.

## The constructor

`SpecFor(instance, mount)` returns the module's embedded declaration with the
instance and the mount set. A host calls it once per agency and catalogue name
and gets one table set for each, the way `mwanachama-backend-forms`'
`SpecFor(instance)` and `mwanachama-backend-agency`'s
`SpecForMount(instance, mount)` already do.

- The instance is the agency's slug; the mount is the catalogue's name.
- An empty mount is the default mount, `spec.DefaultMount`.
- Both are held to the spec engine's alphabet by `ParseSpec`, so a catalogue's
  display name ("Tours & Guides") is not a mount. The host keeps a slug for it.
- Tables are `<instance>_hashOf(<mount>)_hashOf(catalog_<object>)`, so two
  catalogues of one agency and the same catalogue name in two agencies never
  share a table. `TestSpecForGivesEachAgencyAndMountItsOwnTables` migrates two
  mounts of two agencies into one SQLite database and checks every table is
  distinct and created.

## Why the agency example is the base

Before CAT60 this module shipped no domain declaration of its own, only the two
examples under `spec/examples/`. Forms and agency each embed one, so
`SpecFor` needed a base.

`agency.catalog.json` is the smallest honest one: it is the only shipped
declaration that names all fourteen roles (the library example leaves four
out), it is the shape wakala-api's `wakala.catalog.json` already copies, and it
is already run by the whole test suite through `newManager`. Embedding it
rather than writing a third copy keeps one declaration to drift, not two.

Two costs, recorded rather than hidden:

- **The example is now also configuration.** That is the CAT6 complaint, and
  this makes the agency example live config for every per-agency catalogue.
  A dedicated embedded declaration (for instance `catalog.catalog.json`) would
  separate the two; it is not built, because what an agency's own catalogue
  should call its objects is an owner's decision, not this module's.
- **Its two entry indexes are the storefront's** (`_template.sector`,
  `_template.archetype`). They are indexes over a document path, so they cost
  an agency's catalogue an unused index each and nothing else; no column and no
  stored value carries the word.
