# QA

Test coverage and results for `mwanachama-backend-catalog`. The tests live
beside the code they cover; there is no separate test corpus.

```sh
go test ./...        # in-memory SQLite (glebarez/sqlite), no database needed
make test-pg         # -tags=integration, with POSTGRES_URL set
```

As of 2026-09-26 (after `spec/` moved to `mwanachama-backend-shared`, S18,
and this repo's own store/blueprint/manager were refactored onto it,
commit 37e0ae7): 81 test functions offline, all green — `entry_test.go`
(14), `suggestion_test.go` (12), `routes/tools_test.go` (10), `run_test.go`
(8), `examples_test.go` (8), `routes/routes_test.go` (7), `sharelink_test.go`
(5), `private_test.go` (5), `store_test.go` (4), `domain_agnostic_test.go`
(3), `blueprint_test.go` (3), `validate_test.go` (2), plus 7 more in
`postgres_integration_test.go` that need a database. `spec/` now holds only
`examples/` — its own loader/validator/migrator tests moved with the code to
`mwanachama-backend-shared/spec`'s own suite. SQLite in memory is the
expected way to verify a change here — reach for a real Postgres only for
what the section below lists.

## What is guarded, and how

- **The behaviours in [CLAUDE.md](../../CLAUDE.md)** each have a test, and
  the first two were mutation-checked rather than trusted: leaking a pending
  entry from the identical-404 refusal fails the test, and turning the stall
  counter's `>=` into `>` fails it.
- **The domain-agnostic rule** is enforced, not asserted:
  `domain_agnostic_test.go` parses `models/`, `routes/` and the root package
  and fails on any identifier carrying a domain word, with a second test over
  every stored enum value — because a stored value outlives a rename. It was
  mutation-checked by reintroducing `Entry.Sector`. (`gormstore/` was
  deleted 2026-09-23, CAT1 — see CLAUDE.md's "What is superseded".)
- **The spec's refusals** — bad names, an underscore in a segment, a
  duplicate role, identifiers the database would merge past 63 bytes, a
  required field carrying a default, unknown JSON keys — are each their own
  test in `spec/spec_test.go`. The truncation and budget tests were
  mutation-checked by widening `MaxIdentifier` and by restoring the looser
  pattern on the segments.
- **Two domains in one database**: `TestTwoDomainsCoexist` migrates both
  example specs into one database; `TestMigrateIsIdempotent` runs a migration
  twice.
- **A visibility a domain adds stays closed to anonymous readers.**
  `widened_visibility_security_test.go` (security sweep, 2026-10-02) widens
  the blueprint's `visibility` values with `embargoed` — the S37 widening a
  domain spec is allowed — sets an entry to it, and asserts it opens for
  neither a keyless nor a keyed reader (the same `ErrEntryNotFound` text as a
  missing slug), never appears in `ListPublicEntries`, and cannot be given a
  share key. `OpenEntry` is an allow-list (`public` opens, `private` opens
  with a live key, anything else is not found); mutation-checked by turning
  its `default` arm into "open", which turns the test red.
- **The anonymous page-view door cannot grow its table.**
  `routes/pageview_cost_security_test.go` (security sweep, 2026-09-28) sends
  800 anonymous `POST /page-views` across 402 distinct slugs — never-filed
  entry slugs, a pending entry, and index views carrying a slug — and asserts
  exactly two counter rows remain (the index and the one openable entry),
  each at 200. The live probe that prompted it: 6,000 anonymous calls in
  428 ms, `204`×4,000 / `404`×2,000, two rows. Mutation-checked by removing
  `RecordPageView`'s index-slug normalisation.
- **A malformed `limit` never reaches the store.**
  `routes/runs_limit_security_test.go` (security sweep, 2026-09-28) sends
  `GET /runs?limit=` with an overflowing integer, `abc`, `1e3` and a NUL, and
  asserts each answers `400` naming only the argument
  (`limit: "abc" is not a int`) — no driver, SQL or `strconv` detail. What it
  does not pin: `limit=0` and `limit=-1` both return every run, and a raw `;`
  in the query makes `net/http` drop the whole query string, so
  `?limit=1;x` also returns every run. All three are the unbounded-list cost
  CAT10 already tracks, not a separate door.
- **A private-parts write lands on the entry the path names.**
  `routes/private_slug_security_test.go` (security sweep, 2026-09-29) sends
  `PUT /entries/alpha/private` with a body naming `"slug":"beta"` and a forged
  `updated_at`, and asserts beta's readme and rank are untouched, the write
  lands on alpha, and the clock is the server's; the same body under a missing
  slug answers `404` and changes nothing. `put_private` binds the whole body and
  relies on the dispatcher's `into: "Slug"` path overwrite — mutation-checked
  by disabling that overwrite in `mwanachama-backend-shared/dispatch`.
- **A filed suggestion's lifecycle fields are the server's.**
  `routes/suggestion_forged_fields_security_test.go` (security sweep,
  2026-09-29) files a suggestion whose body claims `status: built`,
  `attempts: 42`, an `id`, a 1999 `created_at` and a `decided_at`, and asserts
  it reads back `proposed` with no attempts, no decision, the server's id and
  clock (the claimed id answers 404); fulfilling it while proposed and
  deciding it `stalled` both answer 409 and leave it proposed.
  Mutation-checked by dropping `FileSuggestion`'s attempts reset. The same run
  measured the comment door: 5,001 comments on one suggestion read back as
  1,822,058 bytes in 51 ms with `?limit=1` ignored, and the library accepts a
  2 MiB comment body — CAT10's unbounded-list class behind the
  `catalog.suggestion.comment` action; a mounting host's body cap is the only
  bound.
- **Every gated door asks for its own action before it acts.**
  `routes/deny_all_security_test.go` (security sweep, 2026-09-29) mounts every
  gated operation twice — as a REST route through `OperatorRoutesFor` and as
  an MCP tool through `ToolsFor` — under a `Mount` whose authorizer records
  what it is asked and refuses everything, over real seeded ids (an entry, a
  private entry with a live share link, a proposed suggestion) so a refusal
  cannot be a 404 in disguise. Each of the 24 doors must answer
  `403`/`dispatch.ErrForbidden`, must have asked for exactly its own declared
  action, and must leave the entry, suggestion, share link, runs, settings and
  comments unchanged. Iterating the built set rather than a list means an
  operation added later is covered on arrival. Mutation-checked by skipping
  `handlerFor`'s gate for `catalog.setting.write` (`PUT /settings/k` answered
  `200` and wrote the setting) and `invokerFor`'s for `catalog.run.record` in
  `mwanachama-backend-shared/dispatch`.
- **A flood of repeated statuses that breaks the store names nothing about it.**
  `routes/repeated_status_flood_security_test.go` (security sweep,
  2026-09-29) sends `GET /suggestions` with `status=proposed` repeated 40,000
  times (640,000 bytes, under `net/http`'s 1 MiB header limit) and calls
  `catalog_suggestion_list` with the same 40,000-element list. `status` is the
  only `repeated` argument any of the four libraries declares, and each value
  becomes one bound parameter, so the store's query passes SQLite's variable
  limit (`SQL logic error: too many SQL variables (1)` from the manager
  directly). That failure is the test's precondition; what it pins is that
  neither door then says anything about the database. Both answer the
  dispatcher's opaque `internal error` (a `500`, ~48 ms) — the status is not
  pinned. Mutation-checked
  by writing the call's own error in place of the opaque one in
  `mwanachama-backend-shared/dispatch`'s `handlerFor` and `invokerFor` — each
  half then fails naming `SQL`. Postgres's limit is 65,535 parameters, so the
  same flood succeeds there and costs one 40,000-term `IN` list per request;
  unmeasured (no Postgres plane in the run).
- **An MCP call cannot smuggle a server-owned field in another casing.**
  `routes/mcp_hidden_field_security_test.go` (security sweep, 2026-09-30)
  invokes `catalog_suggestion_file`, `catalog_entry_upsert` and
  `catalog_run_record` with `Status`, `ATTEMPTS`, `Created_At`, `Visibility`,
  `Approved_At`, `SubmittedBy`, `Submitted_By` and a `Slug` shadowing the
  declared `slug`. `encoding/json` matches keys case-insensitively, so a
  dispatcher that bound arguments with `DisallowUnknownFields` would accept
  every one; `mwanachama-backend-shared/dispatch`'s `invokerFor` compares
  each key exactly against the declared arguments and refuses the call by
  name, and nothing is stored — the same calls without the forged key store a
  `proposed` suggestion with zero attempts and a `pending`, unapproved entry.
  This holds whether or not the MCP host validates the tool's
  `additionalProperties: false` schema. Mutation-checked by letting
  `invokerFor` accept a key whose lower-cased form is declared: the `Slug`
  call then stores entry `alpha`.
- **`routes/routes_test.go`** is kept deliberately as the acceptance test the
  W14 dispatcher must pass **unchanged**, even though the package it
  currently exercises is superseded.

## What only a real Postgres covers

`postgres_integration_test.go` is tagged `integration` and skips unless
`POSTGRES_URL` is set, so `go test ./...` stays offline. Seven tests, last
run green against the live `mwanachama-2` project (PostgreSQL 17.6):

- **`doc` is really `jsonb`** there and `text` on SQLite, read back off
  `information_schema` rather than assumed.
- **A document filter goes through `doc #>> '{_template,sector}'`** against a
  real jsonb column, the operator SQLite never sees.
- **Every declared path index is accepted** and appears in `pg_indexes` — and
  the companion test removes the inner parentheses from the emitted DDL and
  asserts Postgres *rejects* it. That is the database mutation-checking the
  double parens, which `TestExpressionIndexesAreDoubleParenthesised` can only
  assert about a string.
- **Identifier truncation at 63 bytes is silent**: two names agreeing that far
  are created as one relation, carrying only the first one's columns. This is
  the premise `spec.Validate` refuses on, and until now it was assumed.
- **A document survives jsonb semantically** but not byte-for-byte, which is
  what the review comparison had to learn.

Each run mints a random instance sized to the spec's own identifier budget,
so its tables are `pg<hex>_catalog_*` and are dropped on cleanup.

**That override is load-bearing, not tidiness.** Since 2026-09-23 the live
`mwanachama-2` database holds `agency_catalog_*` — 634 entries, seeded from
the spec, with the older bare `agency_*` tables dropped after a row-level
parity check. Those are exactly the names
`spec/examples/agency.catalog.json` emits unmodified, at instance `agency`
and module `catalog`. So anything that loads the shipped spec as-is and
points at that database is pointed at production. Override the instance, and
never re-seed or truncate it: ten rows from a 2026-09-21 mine-agencies rescue
run and our own `mwanachama-wakala` agency exist there and in no folder on
disk.

Three bugs so far were found by a real database rather than by the SQLite
tests — the missing inner parentheses, a slug beginning with a digit being
dropped by an identifier pattern, and the byte-wise document comparison in
`UpsertEntry` that jsonb's normalisation turned into a silent un-publish
(CAT2). That is the standing argument for keeping this suite runnable.

`cmd/import` has still never been run against a live database; `cmd/seed`'s
SQL path has (623 agencies into `mwanachama-2`).

## First live probe through a real mount (2026-09-26)

Until this date, this module's routes had only ever been exercised as a bare
library (`CatalogManager` called directly, as CAT10's own row above does) —
`mwanachama-wakala-api`'s mount (`buildCatalog`, `cmd/server/catalog.go`)
never calls `spec.Migrate`, so a freshly created Postgres database 500s on
every `/catalog/...` request until the spec's DDL is applied by hand (see
that repo's own `documentation/4. qa/README.md`). The 2026-09-26 scheduled
security sweep did that by hand once (a throwaway local Postgres, `cmd/ddl`
piped to `psql`) and drove the real router for the first time. Confirmed
live: `GET /entries/{slug}` (anonymous) refuses a never-registered slug, a
never-published (pending) Entry's own slug, and a private Entry's slug with
byte-identical `{"error":"entry not found"}` 404 bodies — the flagship
"Behaviours to preserve" invariant in CLAUDE.md, previously only proven at
the library level. Also confirmed: every operator route (`PUT`/`DELETE
/entries/{slug}`, `.../visibility`, `.../admin`) is reachable by any
authenticated caller and by none anonymously, and — because this module
"names no domain" and therefore has no submitter/ownership check of its own
— any two authenticated operators can fully read, overwrite, publish or
delete each other's Entries. That last part is not a defect: it matches
`mwanachama-wakala-api`'s own `documentation/2. design/tenancy-and-identity.md`
verbatim ("The catalog is outside it... gated on being a user of the
project, not on membership of an Instance... If the catalog ever needs its
own operator/reader split, it needs its own rule").

## Red by design

`go test ./...` is red for exactly thirteen tests (re-counted by the security
sweep 2026-10-01T22:30Z, fifty-fourth run: the integration sweep's CAT31 had
joined the twelve the previous count named — CAT22/CAT23, the owner's own
CAT27 and the sweep's CAT29/CAT30 on top of the original seven):

- the integration sweep's `TestCAT22_OpenBug_SearchTreatsUnderscoreAndPercentAsWildcards`
  (CAT22) and `TestCAT23_OpenBug_ConcurrentPutSettingOnANewKeyFailsInstead`
  (CAT23) — see their board rows;
- the integration sweep's `routes/cat31_attach_status_test.go`'s
  `TestCAT31_OpenBug_RejectedArrivalStillCountsAsAttached` (CAT31) — a
  rejected arrival still counts as attached to a board item; see its row;
- `routes/municipality_reports_test.go`'s `TestMUNI6_ANoteSaysWhoAnsweredIt`,
  red by the owner's own design in 92a8e28 until CAT27 (a comment's `author`
  is the module's `run`/`admin` enum) is decided;
- the security sweep's `routes/public_filing_security_test.go` —
  `TestCAT29_OpenHole_AnonymousFilerSetsItsOwnQueuePriorityAndSource` (CAT29,
  P1) and `TestCAT30_OpenHole_AnonymousFilingTellsAHiddenBoardItemFromAMissingSlug`
  (CAT30, P1), filed 2026-10-01 (fifty-third run). Once a mount opens
  `catalog.suggestion.file` to the public, an anonymous filer keeps the
  `priority` (jumping the moderator queue at −2³¹), `source: owner` and
  `run_id` it sent, and the dedup answers `409` for a pending or private
  entry's slug where a free one gets `201` — the enumeration `open_entry`
  deliberately hides. Both were first shown through `mwanachama-wakala-api`'s
  booted `/reports` mount and go green when an anonymous filing keeps
  neither field and answers every collision like a free slug;

- the integration sweep's `TestCAT10_OpenBug_ListEntriesHasNoDefaultCap`
  (CAT10) in the root package;
- the integration sweep's `TestCAT20_OpenBug_RankCannotReturnToUnranked`
  (CAT20) in the root package, filed 2026-09-30. `catalog.blueprint.json`
  declares `entry_private.rank`'s zero as the meaningful "unranked, sorts
  last" state, but `PutPrivate`'s merge (`private.go`) treats a zero `Rank`
  the same as an omitted one (`if p.Rank != 0`), so a caller can never
  write an entry back to unranked once it has held any other rank —
  demonstrated against the real manager (`PutPrivate({Rank: 3})` then
  `PutPrivate({Rank: 0})` still reads back `Rank: 3`). Goes green once
  `PutPrivate` can tell "the caller omitted Rank" apart from "the caller
  asked for Rank zero";
- the security sweep's `routes/open_entry_timing_security_test.go` —
  `TestCAT16_OpenHole_PrivateSlugCostsTheStoreMoreThanAMissingOne` (CAT16,
  P1), filed 2026-09-30. The anonymous `GET /entries/{slug}?k=…` answers a
  private slug, a pending one and a missing one with the same 404 body, but a
  private slug given any key costs a second query (`keyOpens`). Through
  `mwanachama-wakala-api`'s `/catalog` mount on Postgres that measured 694µs
  against 467µs (medians of 3,000), and 50 paired requests per slug told
  private from not-private 120 times out of 120. The test counts store
  statements per refusal — today 2 / 1 / 1 — and goes green when every
  refusal does the same work.
- the security sweep's `sharelink_race_security_test.go` —
  `TestCAT17_OpenHole_KeyIssuedWhileLeavingPrivateSurvivesTheRetirement`
  (CAT17, P2), filed 2026-09-30. A share key issued while the entry is being
  made public is inserted after `SetVisibility` has retired the entry's keys,
  so it stays active and opens the entry again once it is private (25/100
  through `mwanachama-wakala-api`'s `/catalog` mount on Postgres). The test
  forces the interleaving with a create callback and goes green when issuing
  a key and leaving `private` can no longer overlap.
- the security sweep's `entry_visibility_race_security_test.go` —
  `TestCAT18_OpenHole_UnchangedResendUndoesAReviewersTakedown` (CAT18, P1),
  filed 2026-09-30 (thirty-eighth run). An unchanged re-send through
  `UpsertEntry` writes back the visibility it read, so a reviewer's
  `SetVisibility(private|pending)` landing between the read and the write
  is undone and the entry opens anonymously again: 91/200 and 81/200 on
  Postgres, 46/50 against a re-send loop. The test forces the interleaving
  with an update callback and goes green when an unchanged re-send stops
  writing `visibility`/`approved_at`.
- the security sweep's `suggestion_attempt_race_security_test.go` —
  `TestCAT19_OpenHole_FailedAttemptReportRevivesARejectedSuggestion`
  (CAT19, P2), filed 2026-09-30 (thirty-eighth run). `RecordSuggestionAttempt`
  writes back the status and count it read, so a reject landing in between
  is undone (47/200 on Postgres) and twelve concurrent failures leave the
  count at 2–4, never stalled (20/20). The test forces the interleaving with
  an update callback and goes green when the report's write is conditional
  on the suggestion still being approved.
- the security sweep's `routes/suggestion_fulfil_race_security_test.go` —
  `TestCAT21_OpenHole_FulfilUndoesAReviewersReject` (CAT21, P2), filed
  2026-09-30 (forty-first run). `FulfilSuggestion` writes `built` over
  whatever status the row holds by then, so a reject landing between its
  check and its write is undone after the reviewer was answered `rejected`
  (83/136 through the route table on shared-cache sqlite). The test forces
  the interleaving with an update callback and goes green when the `built`
  write is conditional on the suggestion still being approved or built.

CAT12's whole class is closed. The two reds the eleventh sweep filed
(2026-09-27, late evening) were cleared 2026-09-29, and each test stays in
place as the regression guard for its fix, with the board id kept and the
`_OpenHole_` infix dropped. Each builds the routes *and* the MCP tools
through `routes.RoutesFor`/`routes.ToolsFor` with a `Mount` whose `Caller`
answers `the-signed-in-caller`, then drives both doors:

- `routes/run_submitter_security_test.go` —
  `TestCAT13_RunSubmitterIsNeverTakenFromTheRequestBody` (CAT13, P1).
  `POST /runs` used to store the body's `submitted_by` verbatim, and so did
  the `catalog_run_record` tool. `record_run` now binds it
  `from: "caller"` `into: "SubmittedBy"`, so the tool refuses a
  `submitted_by` argument and every stored run carries the mount's caller.
- `routes/entry_submitter_security_test.go` —
  `TestCAT14_EntrySubmitterIsNeverTakenFromTheRequestBody` (CAT14, P1).
  `PUT /entries/{slug}` used to store the body's `submitted_by` as the
  revision's author — the attribution a reviewer reads before publishing.
  `upsert_entry` now carries the same caller binding beside its `slug`
  overwrite: the address and the caller outrank the body, nothing else does.

Both were mutation-checked — reverting the one declaration in
`catalog.operations.json` puts each back in the red.

Pinned green since that sweep: `routes/sharelink_caller_security_test.go`'s
`TestCAT12_AMountedShareLinkRecordsTheMountsCallerAndMCPRefusesAnIssuer` —
with a mount supplying a caller, a share link issued over REST records that
caller whatever the body's `by` says, and `catalog_share_link_issue` refuses
a `by` argument outright (`takes no argument "by"`).

The two reds before those were cleared 2026-09-27, on the same terms:

- `routes/pageview_security_test.go` —
  `TestCAT11_PageViewAnswersAnUnpublishedSlugLikeAMissingOne` (CAT11, P1).
  The anonymous `POST /page-views` used to answer `204` for a pending or
  private slug and `404` for one never filed, an enumeration oracle for
  exactly the slugs `GET /entries/{slug}` refuses to confirm. `RecordPageView`
  now checks the slug through `OpenEntry(ctx, slug, "")`, so the
  identical-404 invariant above holds on this door too and a view counts only
  against an entry an anonymous reader could open.
- `routes/sharelink_security_test.go` —
  `TestCAT12_ShareLinkIssuerIsNeverTakenFromTheRequestBody` (CAT12, P2). A
  share link's `created_by` used to be whatever the request body's `by` said.
  `issue_share_link` now binds `by` from `from: "caller"` —
  `mwanachama-backend-shared`'s `dispatch` gained that source and a
  `Deps.Caller`, and `routes.Mount` is where a host passes one. The module
  still holds no auth model: with no `Caller` supplied the issuer is `""`,
  never the request's.

Both are pinned at the mount as well, in `mwanachama-wakala-api`'s
`internal/api/http/catalog_issuer_security_test.go` — which is where the
end-to-end proof has to live, since `make self-test` mounts no catalog
(`buildCatalog` needs `CATALOG_DATABASE_URL` and `CATALOG_SPEC`, and this
family does not stand up a real Postgres to test).

Probed and holding on the same run, through `mwanachama-wakala-api`'s
router: a share key opens only the entry it was issued for (another private
entry answers the same 404), a key dies when its entry leaves `private` and
does not revive when it returns, and `PUT /entries/{slug}` ignores a body's
`visibility`, `approved_at`, `id` and `created_at` — a changed document
still lands `pending`. Those are pinned in wakala-api's
`catalog_storefront_security_test.go`.
