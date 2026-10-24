# Agency catalogues declared in the definition

CAT130, from the member app's decisions 399 and 400 (owner, 2026-10-24).

## The declaration

`SpecFor(instance, mount)` now embeds `agency.catalogue.json`, a declaration
of its own for an agency's catalogues, instead of the storefront example.
It keeps every object name the example used (`agency`, `agency_private`,
`share_link`, …) and the domain `agency`, so every table a per-agency
catalogue already holds keeps its physical name and nothing moves. It drops
the storefront's two entry indexes over `_template.sector` and
`_template.archetype`. `spec/examples/agency.catalog.json` stays the shop's,
as CAT6 asked, and wakala-api's `wakala.catalog.json` is unchanged.

This supersedes the "Why the agency example is the base" costs recorded in
[catalogue-per-agency.md](catalogue-per-agency.md).

## The `catalogues` setting

The blueprint declares a `json` setting, `catalogues`, defaulting to an empty
list: every catalogue an agency has, each

| Key | Meaning |
| --- | --- |
| `mount` | the catalogue's mount, in the spec engine's segment alphabet; unique in the list |
| `name` | the display name, required |
| `audience` | `public`, `members` or `staff`, the same values as `Audience` |
| `services` | true when the catalogue holds services (decision 399), so the host checks a service's terms on it |

The forms catalogue and the ideas board are built in (decision 400) and never
appear in the list.

## Checked at publish

`CheckSettings(raw json.RawMessage) []string` is the catalog module's half of
agency's `SettingsChecker` (AG83): the host calls it for module `catalog` with
the module's settings from the draft. It refuses a setting the blueprint does
not declare, a value of the wrong type (shared's `spec.CheckSettings`), and in
`catalogues` an unknown key, a bad or repeated mount, an empty name or an
unknown audience. Each problem is one string, which agency turns into one
blocking issue. Empty settings pass. `ParseCatalogues(value)` reads the list
for a host that mounts what a published definition lists (wakala-api WK188).

Agency needed no change: its checker is already per-module and host-supplied.
