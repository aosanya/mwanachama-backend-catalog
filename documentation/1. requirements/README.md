# Requirements

A catalog of documents with a review state: things are proposed, built,
reviewed, published or shared under a key, and counted. A Go library backed
by Postgres through GORM, imported directly by whatever mounts it — no
service, no gRPC — the same shape
[mwanachama-backend-actor](../../../mwanachama-backend-actor),
[mwanachama-backend-assetmanager](../../../mwanachama-backend-assetmanager)
and [mwanachama-backend-forms](../../../mwanachama-backend-forms) already
took.

## The problem

The mined agency catalog — 623 agencies, 21.3 MB of documents, plus their
module configs and readme/sales files — was held in 11 hand-written tables
and ~33 `security definer` functions that existed only in
`mwanachama-website/supabase/migrations/`. No Go module, no reuse, and one
`.sql` file testing all of it.

Two existing modules were measured against it before a new one was written,
and neither could hold it:

- **forms** models the lifecycle *better* than that SQL does — `PublicLink`
  keeps key, status and `retired_at` where the SQL overwrote a single hash,
  and `Approval` is a real submit/withdraw/approve/refuse. But a Form's
  content is its Questions: no model has a JSON, blob or payload field.
- **assetmanager** maps a slug onto `Code` and a grouping onto `Category`
  cleanly, then has nowhere for a 34 KB document but `AttributesJSON`, a
  plain `text` column its own comment calls a v1 stand-in — and carries
  `Movement` and `Hold`, which a catalog never uses.

`ShareLink` and the review state are modelled on forms's shapes rather than
reinvented.

## What it refuses to know

**A word that means something in one domain and nothing in another does not
belong in this module.** "Sector" fails it — a library has genres. "Entry"
passes it. The module was written for an agency storefront and must run a
library or a course catalog without a line changing; a domain's own
vocabulary lives inside the document and is filtered by JSON path, or is
declared as the domain's own object in its spec.

The full rule, the three things that failed it, and the behaviours that must
survive any refactor are in this repo's [CLAUDE.md](../../CLAUDE.md) and
[doc.go](../../doc.go). `domain_agnostic_test.go` enforces it rather than
asserting it.

## What it deliberately leaves out

**Who may do any of this.** The SQL this replaces keyed its admin table off
Supabase Auth's `auth.users`. This module has no auth model, matching
assetmanager: a route still needs a capability gate wrapped around it by
whatever mounts it, and the role→action grant table belongs in
`mwanachama-backend-auth` — see
[architecture-spec-driven-modules.md](../../../mwanachama-developer/documentation/2.%20design/architecture-spec-driven-modules.md)
§5.

## Where the product decisions are taken

The programme rows — W13 (build the module and move the shop store onto it),
W14 (the spec-driven dispatcher), W15 (the shop served through wakala-api),
W16 (the `<agency>_<module>_<object>` naming decision) — are on
[`mwanachama-website`'s board](../../../mwanachama-website/documentation/3.%20implementation/todo.md),
because they are decisions about the product rather than about this module.
This repo's own board holds the module's own work.
