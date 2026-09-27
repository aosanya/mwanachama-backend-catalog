# mwanachama-backend-catalog — documentation

## Layout

Four folders, in SDLC order, and everything lives under one of them.

| Folder | What's inside |
|--------|---------------|
| [1. requirements/](1.%20requirements/) | Why this module exists and what it refuses to know. The full rule and the behaviours it protects stay in [CLAUDE.md](../CLAUDE.md) and [doc.go](../doc.go). |
| [2. design/](2.%20design/) | What is true of this repo today. The spec-driven shape itself lives one level up, in [architecture-spec-driven-modules.md](../../mwanachama-developer/documentation/2.%20design/architecture-spec-driven-modules.md), because it is an org-wide shape this module is only the first to take. |
| [3. implementation/](3.%20implementation/) | The work: `todo.md` (open board), `todo_done.md` (completed rows + board context). |
| [4. qa/](4.%20qa/) | How this module is verified, what is mutation-checked, and what only a real Postgres can cover. The tests themselves sit beside the code. |

## Where this module is tracked

Two boards, deliberately, and the split is by who the reader is:

- **This board** holds the module's own work — retiring `gormstore/`,
  shrinking `models/`, the tests it still owes.
- **`mwanachama-website`'s board** holds the programme rows that happen to
  land here: W13 (build the module and move the shop store onto it), W14
  (the spec-driven dispatcher), W15 (the shop served through wakala-api) and
  W16 (the `<agency>_<module>_<object>` naming decision, which spans
  wakala-api's registry too). They are decisions about the product, taken
  where the product is tracked.

A row is stated in one place and pointed at from the other.
