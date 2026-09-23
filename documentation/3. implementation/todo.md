# mwanachama-backend-catalog (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in [todo_done.md](todo_done.md).

Programme rows that land in this module — W13/W14/W15/W16 — are on
`mwanachama-website`'s board; see [documentation/README.md](../README.md) for
why the split runs that way.

| Task | Title | Status | Notes |
|------|-------|--------|-------|
| CAT2 | 🧪 **The Postgres integration test this module still owes.** | 🚀 | Everything is verified on SQLite (`glebarez/sqlite`), which is the convention here and the right default — but three behaviours only exist on Postgres and are currently unguarded. The `json` column type is `jsonb` there and `text` on SQLite (`spec/migrate.go`'s `columnType`), so every document read goes through a different operator: `doc #>> '{a,b}'` against `json_extract(doc, '$.a.b')`. An expression index needs its own inner parentheses on Postgres — `on t ((expr))`, a syntax error without them — and SQLite accepts one pair, so a unit test cannot catch a missing one; `TestExpressionIndexesAreDoubleParenthesised` asserts on the emitted string rather than on a database accepting it. And identifier truncation at 63 bytes, which CAT-side validation now refuses at load, is a Postgres rule SQLite does not share. Gate it behind a build tag like the sibling repos do, so `go test ./...` stays offline. Carried over from the website's W13 "still to do". |
| CAT4 | 🗑️ **Delete `routes/` once the dispatcher exists.** | ⏸️ | Twelve hand-written handlers the owner rejected; CLAUDE.md says plainly **do not copy this package**. It cannot go yet — it is the module's only HTTP surface until the website's W14 dispatcher lands, and `routes/routes_test.go` is that dispatcher's acceptance test, which must pass **unchanged** against it. So: keep the test, delete the handlers, and only when there is something else to run the test against. Blocked on W14. |

_Completed rows move to [todo_done.md](todo_done.md)._
