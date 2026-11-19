# Hub import drift

Before the hub imports this module and deletes its in-tree copy (`projects/hub-steward/code/wakala-api/modules/catalog`), the two sides must agree. This page lists every difference found on 2026-11-19, each with what was done about it. The method is `scripts/hub-copy-diff.sh`, which rewrites the hub's imports to this repository's paths, strips comments and layout, and diffs the two trees (see [hub-copy-diff.md](hub-copy-diff.md)).

The hub copy had moved on since 2026-11-16: it gained four changes this repository lacked, and this repository gained three the hub lacks. The script exited 1 with the differences below. After the merge it still exits 1, and what it prints is only what this repository carries on its own (section 2) plus two cosmetic differences (section 3).

Hub commit hashes below are from the hub's `dev` branch at the time of writing.

## 1. Only in the hub: merged into this repository

| What | Hub commits | Files here | Disposition |
|---|---|---|---|
| An entry carries `kind` and `form_id` inside its document; reads and lists return them; the list narrows by `kind` | 28906e2d, 1933f0a3 | `entry.go`, `entry_kind.go` (`withDocFields`, `putDocFields`, `FormIDOf`), `entry_list.go`, `models/entry.go` (two fields tagged `spec:"-"`), `catalog_manager.go` (`ListFilter.Kind`), `routes/tools.go` (field docs), `catalog.operations.json` (descriptions of `list_entries` and `upsert_entry`), test `entry_service_kind_test.go` | Merged. Neither is a column: both live in the document. |
| A service entry carries an inspector role, a fee amount and days until due, with checks (negative fee and fewer than one day refused, 30 days when unset) | 18682cc7 | `models/service.go`, test `service_permit_terms_test.go` | Merged. |
| One board item is read with its outcome, assignee, hidden flag and tally, hidden or not | c912f0f8 | `board_item.go` (`GetBoardItem`, and the tally count of `ListBoardItems` moved into `tallyOf` so both share it), `catalog_manager.go`, `catalog.operations.json` (`get_board_item`, action `catalog.board_item.read`), test added to `board_cover_test.go` | Merged. |
| Staff list what one person raised | b3a79d78 | `catalog.operations.json` (`list_person_raised_items`, action `catalog.raised_item.list_of_person`), test `routes/person_raised_items_red_test.go` | Merged. `ListRaisedItemsOfPerson` already existed here; only the operation was missing. |
| Staff list one person's bookings | 871a2bc1 | `catalog.operations.json` (`list_person_bookings`, action `catalog.booking.list_of_person`), test `routes/person_bookings_test.go` | Merged. `ListHolderBookings` already existed here and serves `list_my_bookings`. |
| Tests that exist only in the hub | | `entry_service_kind_test.go`, `service_permit_terms_test.go`, `routes/person_bookings_test.go`, `routes/person_raised_items_red_test.go` | Moved with the code, imports rewritten. |

## 2. Only in this repository: kept, the hub lacks them

| What | Files | Disposition |
|---|---|---|
| A raised item keeps a one-way sender fingerprint per period, and a second wish from the same person in the period is refused with `ErrWishAlreadySentThisPeriod` (409) | `catalogue_setting.go`, `raised_item.go`, `errors.go`, `models/raised_item.go`, `catalog.blueprint.json` (`sender_hash`, `sender_period`, the `sender` index), `catalog.operations.json` (error status), `routes/routes.go` (sentinel), `UseSenderSecret` and `UseClock` on `CatalogManager`, and their tests | Kept. The limit is off while no secret is set, so the hub's behaviour is unchanged until it calls `UseSenderSecret`. The hub does not call it today (see owner question 1). |
| The query guard that refuses a request carrying more values than the toolchain's parameter cap | `routes/query_guard.go`, one call in `routes/routes.go`, `routes/oversized_query_refused_test.go` | Kept (hub issue CAT135, CAT136). |
| 137 test files the hub copy never carried (66 in the root package, 5 in `models`, 66 in `routes`), `routes/testdata` and `spec/examples` | | Kept. They test code that is now identical in both. |
| `cmd/ddl`, `cmd/import`, `cmd/seed` | | Kept. The hub never had `cmd/`. |
| The allowance for identifiers naming an agency in `domain_agnostic_test.go` | | Kept. Renaming them is open (owner question 2). |

## 3. Differing beyond the import path, not behaviour

| File | Difference | Disposition |
|---|---|---|
| `board_item_tally_test.go` | The hub's tests build their manager with `entryManager(t)` and a fresh context; this repository's use `newManager(t)`, which returns both. | Kept as here. The same assertions run. |
| `models/board_item.go` | `SharedBoardItem` is declared before `BoardItemTally` in the hub and after it here. | Kept as here. |
| `catalog_manager.go` | `GetBoardItem` sits in a different place in the interface. | Kept as here. |

No operation has two behaviours. Where both sides described the same operation (`list_entries`, `upsert_entry`) the hub's text is a superset of this repository's and was taken.

## 4. Hub-only wiring: not part of the module, not moved

These stay in the hub when it imports this module; they are listed so the import does not lose them.

| Wiring in the hub (`code/wakala-api`) | What it does |
|---|---|
| `internal/api/http/forms_catalog_routes.go`, `forms_catalog_entry_submissions.go` | The forms-facing catalogue routes; they call `catalog.FormIDOf` and wrap the manager in `agencyOwnedCatalog`, which calls `catalog.AsAgencyOwner`. All three exist here. |
| The refusal of an entry whose `form_id` is not a form of the agency (hub commit ca1b3830, `forms_catalog_form_check_test.go`) | Needs the forms mount, so it stays in the hub. The module only stores `form_id`. |
| `modules/agency/models/permissions_catalog.json` | The permission catalogue. `board.manage` carries `catalog.board_item.read`; `person_raised_items.read` carries `catalog.raised_item.list_of_person`; `appointment.view_person` carries `catalog.booking.list_of_person`. The three actions are now declared here too. |
| `internal/api/http/mcp_tools_forms_catalog.go`, `mcp_visible.go`, `mcp_tool_actions.go`, `mcp_roster_pin_test.go` | Tool names (`forms_catalog_*`) and filters over this module's tools. The roster pin counts tools and will need three more. |
| `cmd/server/catalog.go` | Mounts the module through `catalog.LoadSpec` and `catalog.ParseSpec`, which exist here. |
| `internal/accounts/router.go`, `internal/api/http/deps.go`, `serverwide_agency.go`, `operation_routes.go`, `reports_routes.go` | Mounts and routing. |

## Counts changed by the union

| Pin | Was | Now | Why |
|---|---|---|---|
| Tools built in `routes/tools_test.go` | 65 | 68 | `list_person_raised_items`, `get_board_item` and `list_person_bookings`. |

## Owner questions

1. Should the hub switch on the one-wish-per-period limit when it imports this module? It needs a secret passed to `UseSenderSecret`. Until it does, the limit is off in the hub.
2. The identifiers that name an agency (`agencyOwnerKey`, `AsAgencyOwner`, `isAgencyOwner`) are allowed by `domain_agnostic_test.go`. Should they be renamed to a neutral word now that the hub will import them from here?

## Check

`go test ./...` passes on in-memory SQLite with the local shared module. `scripts/hub-copy-diff.sh` prints only the sections 2 and 3 differences.
