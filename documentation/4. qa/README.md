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
- **An oversized slug on the anonymous doors reads as a missing one.**
  `routes/oversized_slug_security_test.go` (security sweep, 2026-10-05) sends
  1 KB, 100 KB and 900 KB slugs to `GET /entries/{slug}` and `POST
  /page-views` (entry and index) and asserts each answers byte-for-byte what a
  never-filed slug does and leaves only the one slug-less index counter. The
  `slug` pattern bounds no length, so the host's 1 MiB body cap is the only
  ceiling; the probe measured ~30 ms at 900 KB, linear, no row growth.
  Mutation-checked by removing `RecordPageView`'s index-slug normalisation.
- **A second anonymous filing on a taken slug neither reveals nor rewrites
  the first report.** `routes/colliding_filing_security_test.go` (security
  sweep, 2026-10-06) files a resident's report carrying a phone number and a
  name in `why`/`proposal`, then files a second anonymous report on the same
  slug, and asserts the second answer carries nothing of the first (id, title,
  `why`, `proposal`) and the first reads back untouched. It deliberately does
  not pin the `409` itself: that status is CAT30's oracle and its fix may
  change it. Mutation-checked with a `FileSuggestion` that overwrites the
  title and returns the existing row on a collision — both halves red.
- **An anonymous filing cannot nest its proposal past the decoder, and a
  deep one never breaks the moderator queue.**
  `routes/public_filing_nesting_security_test.go` (security sweep,
  2026-10-09) files anonymous reports whose `proposal` nests 1,000, 4,000 and
  10,001 objects deep (~80 KB, well under the host's body cap) and asserts the
  10,001-deep one is refused `400` by the declared `json_object` rule, the
  1,000-deep one is stored, and `GET /suggestions` still answers `200` with
  every stored proposal decoding as an object. On sqlite a proposal deeper
  than 2,000 levels answers `500 internal error` and stores nothing — the
  driver's own JSON depth limit, not a declared rule — so the 4,000-deep
  filing's status is deliberately not pinned; Postgres `jsonb` has a
  different ceiling and is unprobed. Mutation-checked by making `json_object`
  accept any text.
- **A mount that widens the anonymous list still gates everything else.**
  `routes/widened_anonymous_mount_gate_security_test.go` (security sweep,
  2026-10-07) builds the routes under a mount opening the four municipality
  actions (`entry.open`, `page_view.record`, `suggestion.file`,
  `entry.list_public`, as wakala-api's `/reports` does) with a deny-all
  authorizer, and asserts exactly those four are anonymous, none of them is
  also built gated, and each of the 25 gated routes asks for its own declared
  action, answers `403` and writes nothing. The existing deny-all pin covered
  the default mount only. Mutation-checked twice in `SplitFor`: building the
  gated half without the mount's authorizer, and opening one extra action.
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
- **A share key cannot be guessed.** `sharelink_key_strength_security_test.go`
  (security sweep, 2026-10-08) issues 400 keys for one private entry and
  asserts each opens it, decodes as hex to at least 16 bytes, is never
  repeated, and that every one of the key's positions takes at least eight
  distinct digits across the 400 — a key drawn from a counter, a clock or a
  short seed leaves positions fixed. Mutation-checked both ways: a 6-byte key
  fails the length check, and zeroing the first 28 of 32 bytes fails 56
  positions. Today's key is 32 bytes from `crypto/rand`, hex-encoded.
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
- **A deleted entry's private notes never reach a later entry at its slug.**
  `deleted_slug_private_security_test.go` (security sweep, 2026-10-03) writes
  notes on a private entry, deletes it, and holds that `GetPrivate` answers
  `ErrEntryNotFound` and that whatever re-using the slug does — today
  `ErrEntryExists`, which is CAT37's open bug; after its fix, a fresh entry —
  the old notes are not read back. The notes row is keyed by slug and
  survives the soft delete, so `GetPrivate`'s `findEntry` check is the whole
  control. Mutation-checked by dropping that check: the deleted entry's notes
  come back. Probed first through wakala-api's booted `/catalog` mount
  (`DELETE` → `204`, `GET …/private` → `404`, `PUT` the same slug → `409`).
- **The same holds whatever case the keys are spelled in.**
  `routes/suggestion_forged_fields_key_case_security_test.go` (security sweep,
  2026-10-02) files anonymously on the municipality mount with `Status`,
  `ATTEMPTS`, `Id`, `Created_At` and `DECIDED_AT`. The whole body decodes
  through `encoding/json`, which matches field names case-insensitively, so
  these reach the struct exactly as the lowercase spellings do; the filing
  still reads back `proposed`, with no attempts, no decision, and the server's
  id and clock, because `FileSuggestion` overwrites every one rather than
  relying on a key being absent. Mutation-checked by letting a non-empty
  status through `FileSuggestion`. Probed first through wakala-api's booted
  `/reports/suggestions`.
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
- **A re-send cannot smuggle a changed document past its review.**
  `routes/doc_resend_review_security_test.go` (security sweep, 2026-10-04)
  re-sends a published entry's document over `PUT /entries/{slug}`: reordered
  and respaced it stays on the anonymous read; carrying a second JSON value
  after the reviewed one it is refused `400` and the stored document and
  visibility are untouched; with a key differing only in case, or a number
  written `1.0`, it returns to pending (`404` anonymously). `sameDocument`
  canonicalises with a streaming decoder that would ignore trailing data, so
  what holds the line is `isJSONObject`'s `json.Unmarshal` in `patterns.go`.
  Mutation-checked by swapping it for a `json.Decoder` — the trailing-data
  case goes red.
- **A share key opens only its own entry, in its exact spelling, while live.**
  `routes/sharelink_key_exactness_security_test.go` (security sweep,
  2026-10-04) issues a key for private `alpha` and opens `/entries/alpha?k=`
  with it (`200`), then sends it against a second private entry, upper-cased,
  space-padded either side, NUL-suffixed, truncated by one character, under
  an upper-cased slug, and second behind a wrong `k` (the dispatcher binds the
  first): every one is the same `404` a missing slug answers. A retired key,
  and a key issued before its entry left private and came back, are refused
  the same way. `open_entry` declares no MCP tool, so REST is its only door.
  Mutation-checked by dropping the `slug = ?` predicate from `keyOpens` — the
  second-entry case answers `200`.
- **Retiring a share link again never rewrites when it stopped working.**
  `routes/sharelink_retire_trail_security_test.go` (security sweep,
  2026-10-11) retires one link through `POST /links/{id}/retire` and lets a
  second be retired by its entry going public, waits past the clock's
  one-second grain, then retires both through the route again: each answers
  `200` and its `retired_at` in `GET /entries/{slug}/links` is the one first
  stamped, so the trail a rotation leaves stays honest. Mutation-checked by
  dropping `RetireShareLink`'s already-retired guard — both halves go red.
- **No read answer carries a share key or its stored hash, under any name.**
  `routes/sharelink_hash_value_security_test.go` (security sweep,
  2026-10-05) issues two keys for private `alpha`, opens it with the first,
  then reads `/entries/alpha/links`, a keyed `/entries/alpha?k=`,
  `/entries` and `/entries/alpha/private` and looks for each key and for the
  hex SHA-256 the store keeps — the *value*, so a renamed field cannot slip
  past the way it would past `routes_test.go`'s check on the name
  `key_hash`. Mutation-checked by tagging `ShareLink.KeyHash` `json:"kh"` —
  the link listing goes red on both hashes.
- **A keyed anonymous open carries no operator-only text.**
  `routes/keyed_open_operator_text_security_test.go` (security sweep,
  2026-10-05) writes private notes on private `private-one`, issues it a
  share link whose label names its recipient, opens it anonymously with the
  key (`200`) and looks for both strings in the answer: neither appears —
  the label stays on `/entries/{slug}/links` and the notes on
  `/entries/{slug}/private`, both operator doors. Not mutation-checked.
- **A deleted entry is gone from every door, and a re-send revives nothing.**
  `routes/deleted_entry_doors_security_test.go` (security sweep,
  2026-10-05) gives `blocked-drain` private notes, a share key and an
  approval, deletes it, and walks eight operator doors (each `404`) and
  three anonymous ones — plain open, keyed open with the pre-delete key,
  page view — each answering byte-for-byte what a never-filed slug does.
  A re-`PUT` of the unchanged document is accepted only as a fresh
  `pending` entry with no approval or notes carried across (today it is
  `409`, the integration sweep's CAT37), and the old key never opens it.
  Mutation-checked by making `DeleteEntry` leave `deleted` false — thirteen
  assertions go red.
- **An upsert body never publishes its own entry.**
  `routes/upsert_review_fields_security_test.go` (security sweep,
  2026-10-05) sends `PUT /entries/forged` claiming `visibility: "public"`,
  `Visibility: "public"` and a 2000 `approved_at`: each lands `pending` and
  unapproved, opens anonymously as `404` and stays off `/public/entries`.
  Once published through `set_visibility`, a changed document still
  claiming `public` drops back off the anonymous listing. The
  `catalog_entry_upsert` tool refuses both arguments by name and writes
  nothing. Mutation-checked by letting `UpsertEntry` keep a non-empty body
  visibility on a new entry — the anonymous open answers `200`.
- **Two mounts of one instance never reach each other's rows.**
  `routes/cross_mount_isolation_security_test.go` (security sweep,
  2026-10-05) migrates the agency storefront and the municipality reports
  board into one database under the same instance, `wakala` — the shape
  `mwanachama-wakala-api` deploys `/catalog` and `/reports` in. A public
  entry, a share-keyed private entry, a setting and a page view written
  through the storefront are invisible to every board door, including
  delete, retire and the anonymous public listing; a report filed on the
  board is invisible to the storefront's suggestion reads and decide.
  `page_view` is the one object both domains name alike, so the mount
  segment is all that keeps that table apart. Mutation-checked by giving
  both specs the default mount: the board's `GET /page-views` then lists
  the storefront's `alpha`.
- **An anonymous search never widens the public listing past moderation.**
  `routes/public_search_moderation_security_test.go` (security sweep,
  2026-10-05) seeds a published card beside a pending, a private and a
  published-then-deleted report whose names and summaries carry a marker,
  then searches `/public/entries` anonymously for each marker and for
  words all three share, with and without a `visibility` query. Only the
  published card ever comes back. `ListEntries` writes its search as one
  `LOWER(name) LIKE ? OR LOWER(summary) LIKE ?` clause, which GORM
  parenthesises against `deleted` and `visibility`; mutation-checked by
  splitting it into `.Where(…).Or(…)`, which lists all three unpublished
  reports to an anonymous `?search=summary`.
- **A required body argument left out is refused and changes nothing.**
  `routes/required_body_absent_security_test.go` (security sweep,
  2026-10-05) posts `{}` to every operation declaring a `required` body
  argument — `set_visibility`, `decide_suggestion`,
  `record_suggestion_attempt`, `fulfil_suggestion`, `comment_on_suggestion`,
  `attach_suggestion`, `record_page_view` — and each answers a 4xx, leaving
  the public entry public and the approved suggestion's status, attempts,
  priority and comments untouched. The dispatcher does not enforce
  `required` on HTTP (shared's S35), so each refusal is the manager's own;
  the same omission strips a role in `mwanachama-backend-permissions` (PM14).
  Mutation-checked by dropping `RecordSuggestionAttempt`'s comment check —
  the attempt is counted and a blank comment stored.
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
- **No spelling of a query key widens the anonymous listing.**
  `routes/public_listing_key_spelling_security_test.go` (security sweep,
  2026-10-08) was written after the integration sweep's S54 found that
  `dispatch` binds an untagged query field to its all-lowercase Go name.
  `ListFilter` carries no tags, so `/public/entries` answers to `visibility`,
  `docs`, `search` and `limit`. The test sends `visibility=pending`,
  `Visibility=pending`, `VISIBILITY=private`, a repeated `visibility`,
  `deleted=true`, `include_deleted=true`, `limit=100000` in three casings,
  `limit=-1`, `limit=0` and a repeated `limit`. Each answer lists only public,
  undeleted entries, at most `MaxPublicEntryLimit` of them, and none carries
  `doc`. The control is `ListPublicEntries` overwriting `Visibility`
  unconditionally. Mutation-checked: forcing it only when unset turns three
  queries red.
- **No stored entry reaches a tool's name, description or schema.**
  `routes/tool_list_stored_text_security_test.go` (security sweep,
  2026-10-09, OWASP MCP03 tool poisoning) builds the tool set, upserts an
  entry whose name and summary carry a marker and an injected instruction,
  builds it again over the same manager, and asserts the marker is absent and
  every tool's name, title, description, action and input schema is
  byte-identical. The host registers these tools on its `/mcp`, so a
  description built from a row would reach every other caller's model.
  Mutation-checked: appending the first stored entry's name to a description
  in `BuildToolsFor` turns it red.
- **A tool given `null` for every required argument refuses and writes
  nothing.** `routes/tool_null_required_security_test.go` (security sweep,
  2026-10-12) calls each tool declaring a required argument with every one
  of them `null` — `dispatch` checks presence only and binds the zero value —
  and asserts each refuses and that no entry or suggestion was stored. The
  tools with no required argument (listings, `run_record`) are not probed.
  Not mutation-checked.
- **A REST upsert never takes the id, clock, submitter or `deleted` from
  its body.** `routes/upsert_server_owned_fields_security_test.go`
  (security sweep, 2026-10-12) sends `id`, `created_at`, `updated_at`,
  `submitted_by` and `deleted: true` on a create and again on an unchanged
  re-send of a published entry, and asserts every one is the server's and
  the entry stays on the anonymous listing. `dispatch` hides these fields
  only on the MCP plane (`ReadOnly`); on REST the whole body decodes into
  `Entry`, so `entry.go`'s overwrites are the whole control.
  Mutation-checked: dropping the re-send's `e.Deleted = false` turns it red.
- **A wide anonymous filing and a long anonymous search stay linear.**
  `routes/public_filing_width_cost_security_test.go` (security sweep,
  2026-10-09) files a 50,000-key, ~640 KB `proposal` through the
  municipality mount's anonymous `POST /suggestions` (40 ms when pinned,
  1,000 and 10,000 keys scaling linearly), reads it back whole from the
  moderator queue, and sends a 100,000-character `search` to the anonymous
  `GET /public/entries` (1 ms). Each is bounded at 5 s, which only a
  super-linear regression would reach. Not mutation-checked: it is a cost
  bound, and the host's body cap and WK67 already cover the flood.
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

`go test ./...` is red for exactly twenty-nine tests (re-counted by the
integration sweep pass 2026-10-12, which added CAT47 — `DecideSuggestion`
checks the target status but never the suggestion's own current status, so
deciding a `built` suggestion un-builds it while the Entry and match row
that answered it both survive untouched — on top of the twenty-eight the
same routine counted 2026-10-10, which added CAT46 — `ListFilter` has no
`Offset`, so once a listing is capped — which `ListPublicEntries` always is
— a caller can never reach a second page — on top of the twenty-seven the
same integration sweep pass 2026-10-09 counted, which also added CAT45 — the shared
`listOf` helper applies no cap at all, worse than CAT10/CAT44 since its five
callers have no limit parameter to set even if a caller wanted one — on top
of the twenty-six the same run's own prior count, which added CAT44 —
`ListShareLinks` has no
default cap, the same loophole already filed as CAT10 (`ListEntries`), on a
second table — on top of the twenty-five the integration sweep's prior pass
counted 2026-10-06, which added CAT43 — `AttachSuggestion`/
`FulfilSuggestion` race the same way CAT23/CAT34 already do, on a third
table — on top of the twenty-four the security sweep's ninety-fourth run
2026-10-05 counted, which added CAT42 — a re-sent
unchanged `doc` with a new `name` or `summary` keeps an entry's approval, so
the anonymous card changes unreviewed — on top of the twenty-three the
eighty-fifth run counted 2026-10-04, which added CAT41 — an
anonymous `GET /public/entries` takes any number of stacked `docs` filters —
on top of the twenty-two the eighty-first run counted, which added CAT40 — every
anonymous read of a public or keyed entry, and the public listing, hands the
reader the writing operator's `submitted_by` — on top of the twenty-one the
integration sweep counted 2026-10-03, which added CAT39 — `UpsertEntry`'s
changed-document path can leave `private` without retiring any of its share
keys, no race needed — on top of the twenty the 2026-10-02 third pass counted,
which added CAT38 — `decide_suggestion`
cannot lower a suggestion's priority to zero or below once it has held a
positive one — on top of the nineteen the same day's second pass counted:
CAT37, a deleted Entry's slug sterilised forever by an unconditional unique
constraint even though every read already reports it free — on top of the
eighteen the same day's first pass counted: CAT36 (`UpdateEntry` reaches no
declared operation and clobbers a moderator's approval with no race needed)
on top of CAT33/CAT34 (widening CAT17/18/19/23's read-then-write class into
`PutPrivate`) and CAT35 (widening CAT18 itself to the opposite, approval,
direction), themselves on top of the fourteen the security sweep's
fifty-seventh run counted — CAT32 on top of the thirteen the fifty-fourth run
counted, which was itself CAT31, CAT22/CAT23, the owner's own CAT27 and the
sweep's CAT29/CAT30 on top of the original seven):

- the integration sweep's `suggestion_decide_built_test.go`'s
  `TestCAT47_OpenBug_DecideBuiltSuggestionSilentlyUnbuildsItWhileTheMatchSurvives`
  (CAT47), filed 2026-10-12, picking up a second lead the security-hunt
  routine flagged for this one (its hundred-and-fifty-second run): fulfilled
  a suggestion against an Entry (status `built`), then `DecideSuggestion`
  with `rejected` succeeded (`err=nil`) and read back as `rejected` while
  its match row and the Entry it names both survived unchanged, and a
  follow-up approve-then-fulfil cycle built it a second time.
- the integration sweep's `routes/public_listing_offset_test.go`'s
  `TestCAT46_OpenBug_PublicListingHasNoOffsetSoRowsPastTheCapAreUnreachable`
  (CAT46), filed 2026-10-10, picking up the lead the security-hunt routine
  flagged for this one: seeded and published 150 public entries, page one
  of the anonymous listing held the first 100 (`MaxPublicEntryLimit`), and
  `?offset=100` answered the identical 100 rows rather than the 50 past the
  cap — `ListFilter` declares no `Offset` field at all, so a caller has no
  way to ask for a second page, ever. Goes green once `ListFilter` gains an
  `Offset int` and `ListEntries` applies it;
- the integration sweep's `suggestion_listof_cap_test.go`'s
  `TestCAT45_OpenBug_ListSuggestionsHasNoCapAndNoWayToAskForOne` (CAT45),
  filed 2026-10-09, widening CAT10/CAT44's class to the shared `listOf`
  helper itself: filed 300 suggestions and `ListSuggestions()` returned
  all 300, with no limit parameter on the signature at all to ask for a
  smaller page. `ListSuggestionComments`/`ListSuggestionMatches`/
  `ListEntryMatches`/`ListPageViews` share the identical root cause and
  fix location (`catalog_manager.go`'s `listOf[T]`) as evidence in the
  same row, not four more tests. Goes green once `listOf` applies a
  default cap before `.Find`;
- the integration sweep's `sharelink_listcap_test.go`'s
  `TestCAT44_OpenBug_ListShareLinksHasNoDefaultCap` (CAT44), filed
  2026-10-09, widening CAT10's no-default-cap class to a second table:
  `ListShareLinks` builds its query with no `.Limit` either, so an entry
  with 150 keys issued against it gets all 150 of 150 back from one call.
  Goes green once `ListShareLinks` gets a default cap, ideally the same
  fix CAT10 lands on for `ListEntries`;
- the integration sweep's `suggestion_match_race_test.go`'s
  `TestCAT43_OpenBug_ConcurrentAttachSuggestionOnANewPairFailsInstead`
  (CAT43), filed 2026-10-06, widening CAT23/CAT34's check-then-insert race
  class to a third table: `upsertMatch` (shared by `AttachSuggestion` and
  `FulfilSuggestion`) counts rows for `(suggestion_id, slug)` before
  inserting, so two callers attaching the same brand-new pair at once can
  both see zero and one of them hits `suggestion_match`'s composite
  primary key as a raw SQL constraint failure instead of a successful
  write. Goes green once `upsertMatch`'s insert branch recognises
  `isUniqueViolation` the way `CreateEntry`/`PageView`/`FileSuggestion`'s
  own inserts already do;
- the integration sweep's `entry_upsert_leaves_private_test.go`'s
  `TestCAT39_OpenBug_UpsertLeavingPrivateNeverRetiresItsKeys` (CAT39), filed
  2026-10-03, widening CAT17's "leaving private must retire every live key"
  invariant off its race-only door: `SetVisibility`/`DeleteEntry` both call
  `retireLinksFor` when an Entry leaves `private`, but `UpsertEntry`'s
  changed-document branch forces the same transition straight through
  `replaceEntry` and calls neither — a plain, sequential document re-import
  on a private Entry, with no timing window at all, leaves every key issued
  under the old review live, ready to open the entry again the next time it
  is made private. Goes green once `UpsertEntry` retires links on the same
  branch that forces `VisibilityPending`;
- the integration sweep's `suggestion_priority_floor_test.go`'s
  `TestCAT38_OpenBug_DecideSuggestionCannotLowerPriorityToZeroOrBelow`
  (CAT38), filed 2026-10-02, widening CAT20's "zero overloaded as omitted"
  class to a second field on a second object: `suggestion.priority` is a
  plain int with no declared minimum, ordered lowest-first, but
  `DecideSuggestion`'s `if priority > 0 { updates["priority"] = priority }`
  (`suggestion.go`) silently drops a caller's `0` or negative value instead
  of writing it — the only declared operation that can change a filed
  suggestion's priority can never move one to the front of the queue once
  it has held any positive value. Goes green once `DecideSuggestion` can
  tell "the caller omitted priority" apart from "the caller asked for 0 or
  lower";
- the integration sweep's `entry_delete_slugreuse_test.go`'s
  `TestCAT37_OpenBug_DeletedEntrySlugCanNeverBeReused` (CAT37), filed
  2026-10-02. `DeleteEntry` only sets `deleted = true`; `entry.slug`'s
  `unique: true` is a plain, unconditional field-level constraint, so the
  soft-deleted row keeps the slug forever even though `GetEntry`/
  `OpenEntry`/`ListEntries` all already report it free. `UpsertEntry` — the
  only Entry-creating operation this module declares — falls through to
  `CreateEntry` on that same slug and collides with the invisible row,
  answering `ErrEntryExists` (`409` over `PUT /entries/{slug}`) on a slug
  every read says is open. Goes green once `entry.slug`'s uniqueness moves
  from the field onto a declared index with `not_deleted: true` — the
  mechanism `mwanachama-backend-shared/spec`'s `Index.NotDeleted` already
  ships, unused here;
- the integration sweep's `entry_update_unreachable_test.go`'s
  `TestCAT36_OpenBug_UpdateEntryHasNoOperationAndClobbersAModeratorsApproval`
  (CAT36), filed 2026-10-02. `UpdateEntry` is exported on `CatalogManager`
  but no operation in `catalog.operations.json` calls it, and unlike
  `UpsertEntry` it never carries the existing row's `Visibility`/`ApprovedAt`
  forward before its whole-row `replaceEntry` write — a caller holding an
  earlier `GetEntry` snapshot silently undoes a moderator's already-committed
  `SetVisibility`, sequentially, with no concurrency needed. Goes green once
  `UpdateEntry` either merges those two fields the way `UpsertEntry` does or
  is removed as unreachable;
- the integration sweep's `entry_approval_race_test.go`'s
  `TestCAT35_OpenBug_UnchangedResendUndoesAReviewersApproval` (CAT35), filed
  2026-10-02. The reverse of CAT18 below: a resend of a pending Entry's
  unchanged document writes back the pre-approval `visibility`/`approved_at`
  it read, so a reviewer's concurrent `SetVisibility(public)` (the approval)
  is undone — the entry reverts to pending with `approved_at` erased and stops
  opening anonymously, even though both calls answered success. Goes green
  together with CAT18, from the same fix (`UpsertEntry`/`replaceEntry`
  conditioning its write on the row it read rather than a stale snapshot);
- the integration sweep's `private_race_test.go`'s
  `TestCAT33_OpenBug_ConcurrentPutPrivateOnDifferentFieldsLosesOne` (CAT33)
  and `private_insert_race_test.go`'s
  `TestCAT34_OpenBug_ConcurrentPutPrivateOnANewSlugFailsInstead` (CAT34),
  filed 2026-10-02, widening CAT17/CAT18/CAT19/CAT23's read-then-write class
  into `PutPrivate`: two concurrent writers setting different private fields
  on the same Entry lose one writer's field (CAT33), and two concurrent
  first-time writers on a new slug surface a raw constraint error instead of
  both succeeding (CAT34, the same shape as CAT23). See their board rows;
- the security sweep's `routes/upsert_card_review_security_test.go`'s
  `TestCAT42_OpenHole_ResendWithANewNameOrSummaryKeepsTheApproval` (CAT42,
  P1), filed 2026-10-05. `UpsertEntry` keeps `visibility` and `approved_at`
  whenever `sameDocument` says the `doc` is unchanged and never compares
  `name` or `summary` — the only text the anonymous listing shows. Goes green
  once a changed card returns the entry to `pending` like a changed document;
- the security sweep's `routes/public_doc_filter_cost_security_test.go`'s
  `TestCAT41_OpenHole_AnonymousListingTakesUnboundedDocFilters` (CAT41, P2),
  filed 2026-10-04. `ListEntries` adds one `Where` per `docs=` filter with
  no cap, so an anonymous request carrying 13,000 of them (inside the 1 MiB
  header limit) costs ~800 ms against ~0.15 ms for one and is answered
  `500`. Goes green once a listing refuses more than a handful of document
  filters with a `400` before the query is built;
- the security sweep's `routes/public_read_writer_security_test.go`'s
  `TestCAT40_OpenHole_AnonymousReadCarriesTheWritersIdentity` (CAT40, P3),
  filed 2026-10-04. `OpenEntry` and `ListPublicEntries` return the whole
  Entry, so `GET /entries/{slug}` and `GET /public/entries` carry
  `submitted_by` — on wakala-api, the internal subject of the operator who
  wrote it — to a reader with no account. Goes green once the anonymous view
  drops `SubmittedBy` the way the listing already drops `Doc`;
- the security sweep's `routes/run_list_cost_security_test.go`'s
  `TestCAT32_OpenHole_OneCallerGrowsTheRunListAnswerWithoutBound` (CAT32, P2),
  filed 2026-10-02. `POST /runs` stores a summary of any size up to the
  mount's request cap and `GET /runs` with no `limit` answers every run, so
  one caller's writes make a single read arbitrarily large — 200 reports in
  1.4 s made one `GET /catalog/runs` on wakala-api answer 180 MB. Goes green
  once the default answer is bounded, by a default page, a summary cap or both;

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
