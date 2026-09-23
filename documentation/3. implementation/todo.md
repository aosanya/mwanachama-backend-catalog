# mwanachama-backend-catalog (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

Programme rows that land in this module — W13/W14/W15/W16 — are on
`mwanachama-website`'s board; see [documentation/README.md](../README.md) for
why the split runs that way.

| Task | Title | Status | Notes |
|------|-------|--------|-------|
| CAT4 | 🗑️ **Delete `routes/` once the dispatcher exists.** | ⏸️ | Twelve hand-written handlers the owner rejected; CLAUDE.md says plainly **do not copy this package**. It cannot go yet — it is the module's only HTTP surface until the website's W14 dispatcher lands, and `routes/routes_test.go` is that dispatcher's acceptance test, which must pass **unchanged** against it. So: keep the test, delete the handlers, and only when there is something else to run the test against. Blocked on W14. |

_Completed rows move to [todo_done.md](todo_done.md)._
