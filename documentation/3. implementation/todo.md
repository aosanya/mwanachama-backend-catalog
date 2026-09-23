# mwanachama-backend-catalog (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

Programme rows that land in this module — W13/W14/W15/W16 — are on
`mwanachama-website`'s board; see [documentation/README.md](../README.md) for
why the split runs that way.

| Task | Title | Status | Notes |
|------|-------|--------|-------|
| CAT6 | 🔒 **`spec/examples/agency.catalog.json` is an example and production config at the same time.** | 📋 | It is the file the shipped tests load, the file `cmd/import`/`cmd/seed` default to, and the file anything pointing `CATALOG_SPEC` at a real database would name — and at instance `agency` + module `catalog` it emits exactly the `agency_catalog_*` tables the live `mwanachama-2` project holds, 634 entries seeded 2026-09-23. So loading the shipped example unmodified against that database is operating on production. Nothing warns. This is already load-bearing rather than theoretical: `postgres_integration_test.go` is only safe because it overrides `Instance` to a random `pg<hex>` per run, and that override is one line away from being dropped by someone who reads it as tidiness. Move the production declaration out of `examples/` and leave a genuine example behind — a domain nobody has provisioned. Raised by the session wiring wakala-api, 2026-09-23, while noting that a spec-mounting process inherits the same overlap. |
| CAT7 | 🐞 **`ErrInvalidDocPath` returns 500, not 400.** | 📋 | A caller sending a malformed document path gets an internal error, and because an unmapped sentinel is redacted to `"internal error"` it does not even say what was wrong. It is in neither of the two error tables the deleted handlers carried, so `catalog.operations.json` reproduces that faithfully rather than quietly changing a status while W14 was landing. Fix is one line in the spec's `errors` map. Small, but it is the refusal guarding a SQL-injection surface (`spec.DocPathPattern`), so the caller should be told plainly they sent something invalid. Found 2026-09-23 while mapping the tables for the dispatcher. |

_Completed rows move to [todo_done.md](todo_done.md)._
