# routes/

`routes/` is no longer twelve hand-written handlers. It is an adapter: the
route table is declared in `catalog.operations.json` and built by
[`mwanachama-backend-shared/dispatch`](../../../mwanachama-backend-shared/documentation/2.%20design/dispatcher.md).

Three files remain — `routes.go` (74 lines), `routes_test.go`, and nothing
else. `entry.go` and `suggestion.go`, about 500 lines of decode/call/encode,
are gone.

## Why the operations are their own file

`catalog.operations.json` sits beside `catalog.blueprint.json` at the repo
root, and both are module-level: an operation is the module's act, not a
domain's. `catalog.entry.open` is the same act whether the domain calls its
objects agencies or titles, and the addresses are already in the module's
neutral vocabulary — `/entries`, `/suggestions`, `/page-views`, never
`/agencies`.

They are two files rather than one because they answer to different readers,
and permanently so. The blueprint is what the store and the type check are
built against, in this module's own `spec` package. The operations are what
the dispatcher binds to, and the dispatcher lives in `mwanachama-backend-shared`
and cannot import this module at all. One file would have to be parsed by two
packages that are deliberately unable to see each other. A module that wants
declared objects and no HTTP surface simply has no second file.

## The gate is data now, not four function names

`routes/` used to expose `PublicRoutes`, `EntryRoutes`, `SuggestionRoutes` and
`OperationsRoutes`, and the split between the first and the other three was
the security boundary: the storefront is a static SPA with no credential, so
`PublicRoutes` mounts ungated and the rest sit behind a capability gate.
Mounting `EntryRoutes` without one publishes every pending Entry.

That distinction now lives in `AnonymousActions`, an allowlist of **action
ids**, and `PublicRoutes`/`OperatorRoutes` are selections over it. Two reasons
it is better as data:

- **A mounting process no longer needs this module's Go identifiers.** It asks
  which actions it is willing to serve anonymously. That is what lets a
  process mount a spec rather than a named module.
- **The failure direction is safe.** The allowlist names what is public, never
  what is protected, so an operation added to the spec and not named there
  arrives *gated*. It shows up as a 401 rather than as 634 pending entries on
  the public internet. Under the old four-way split a new route added to
  `EntryRoutes` inherited that group's gate — right, but by luck.

The four-way split itself was not preserved because no mount distinguished
those three groups; all of them sat behind the same gate. Action ids leave
room for an admin tier above an operator one without another refactor.

**`routes_test.go` cannot catch a regression here**, because it mounts every
route into one gateless mux. A route table that cannot be mounted safely
passes it green. That is a gap in the acceptance criterion, not only in an
implementation, and it is why the property is tested in the dispatcher instead.

## Behaviour held exactly

The status codes and the error-to-status table are the union of the two tables
the deleted handlers carried, unchanged. So is the redaction: an error the
spec does not map returns 500 with `"internal error"`, never the sentinel's
own text.

One thing the old handlers did that a naive dispatcher would have lost:
`UpsertEntry` and `PutPrivate` ended `in.Slug = r.PathValue("slug")`. The
address outranks the body, so a caller cannot `PUT /entries/a` with a body
claiming slug `b`. That is the `into` binding, and it applies after the body
is decoded.

**The caller outranks the body the same way.** An actor field — who issued a
share key, who submitted a run report, who wrote an entry revision — is bound
`from: "caller"` with an `into`, so the mount's `Caller` overwrites whatever
the request claimed. Three operations carry it: `issue_share_link`
(`ShareLink.created_by`), `record_run` and `upsert_entry` (both
`SubmittedBy`). The same binding keeps the field out of every MCP tool
schema, so an agent is refused by name rather than ignored. A mount that
supplies no `Caller` binds `""` — never the request's value, which is the
point.

The binding lives in the declared operation, not in the manager:
`CatalogManager.UpsertEntry` stores whatever `SubmittedBy` it is handed. A
host that calls the manager directly from a tool of its own gets none of
this, and has to take the actor from its caller itself. wakala-api's
hand-written `catalog_template_mint` does not, and stores the
`submitted_by` its arguments claim — demonstrated 2026-10-01 and filed
there as WK66.

`ErrInvalidDocPath` is deliberately *not* in the table, because it was not in
the old ones either — it currently falls through to 500 although it is a
caller error. Worth a row of its own rather than a silent change here.

## The same table, as MCP tools

`routes.Tools(cm)` returns the operations as tools for an AI agent, built by
the dispatcher's
[`Tools`](../../../mwanachama-backend-shared/documentation/2.%20design/dispatcher.md)
from `catalog.operations.json`. Nothing here is hand-written: 24 tools, one per
gated operation, named from the action with its dots turned into underscores —
`catalog.entry.upsert` becomes `catalog_entry_upsert`.

**The two reader operations are not tools.** `operatorOnly` drops whatever
`AnonymousActions` names, so `catalog.entry.open` and
`catalog.page_view.record` — a storefront's two anonymous acts — have no tool.
An agent holding a credential reads with `catalog_entry_read`, which does not
need a share key and does not pretend to be a visitor; counting a page view it
did not cause would be a lie in the table.

**The prose a model reads is the blueprint's.** `routes.Fields()` walks
`catalog.Blueprint()` and hands the dispatcher each field's description, whether
it is required, and an enum's permitted values, keyed by the Go type that
carries it. So the description of `doc` in a tool's schema is the same sentence
the column was declared with, and adding a field to the blueprint describes it
in every tool that writes it, with no second edit.

Read-only is derived rather than listed: a primary key, any `timestamp`, and
four fields the store owns outright (`Entry.visibility`, `Entry.deleted`,
`Suggestion.status`, `Suggestion.attempts`). `UpsertEntry` overwrites all four,
so offering them to a model would invite it to believe a value it sent was
kept. Visibility has a tool of its own, and it borrows the same declared values
through `"field": "Entry.visibility"`.

`TestEveryDescribedFieldIsOneItsTypeCarries` holds the role→Go type map to the
types themselves, because that map is the one hand-written thing left in this
path and a rename would otherwise cost only the prose, silently.
