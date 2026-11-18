# Tally by area

Built under hub issue MUN1283 (TD-3656): the catalog half in
`mwanachama-backend-catalog` (commit ab977c1), the host directory in
`mwanachama-wakala-api` (commit 06964cb). A council reads the board as a poll: for each board item, how
many people are behind it, area by area, against how many people live there.
`ListBoardItems` gives two totals and no split, and a Me too holds only
`subject_id`, `holder_id` and `created_at`, so the area of a person who said
Me too is not on the row. This page decides where it comes from. Both halves
are built; no screen calls the tool yet.

## Decision: the area of a person is the host's, read at tally time

Not stored on the Me too. Reasons:

- A stored area goes stale when a person moves, and the tally would keep
  counting them where they used to live.
- It adds a column to the `endorsement` role in `catalog.blueprint.json` and a
  migration to every domain spec on a mount that already holds rows.
- The catalog has no actor dependency and must not gain one. The seam for this
  already exists: `UseAreaResolver` lets the host answer a question the module
  cannot ([raised-items.md](raised-items.md), "A raised item's place").

The seam is one value the host passes once, beside the two resolvers:

| Piece | Shape |
| --- | --- |
| `UseAreaDirectory(AreaDirectory)` | on `CatalogManager`, next to `UseAreaResolver` |
| `AreaDirectory.AreasOf(ctx, personIDs)` | `map[personID]area`, one batch call per tally; a person it cannot place is absent |
| `AreaDirectory.Areas(ctx)` | every area with its population, `[]AreaPopulation{Area, Population}` |

A resolver error refuses the tally, as `AreaResolver` does; the module never
guesses an area. With no directory passed the tally still answers: raisers sit
where their item's stored `Area` puts them, every Me too holder is unplaced, and
no area has a population.

The word is `area`, not `ward`, in every identifier and on the wire:
`raised_item.Area` and `AreaResolver` already say it, and "ward" is a
municipality's word ([scope-figures.md](scope-figures.md), "Domain words").

## Two overturns of the first steer

**Raisers are placed by the directory first, their item's `Area` second.** An
item's `Area` says where the thing is (the hub resolves it from the pin). A Me
too holder's area says where the person lives. Mixing them would put two
neighbours behind the same wish into different areas. Filed items are the other
reason: `raised_by` is then a register entry id (decision 374) the directory
may not know, and the stored `Area` is the only answer left. With several
raised items the earliest one with a non-empty `Area` stands, so a person has
one area per tally.

**Population is not a scope figure.** CAT133 (member app decision 424)
retires `scope_figure`: the ward population lives on actor's groups (ACT34,
`LatestGroupPopulation`). Building the tally on a role about to be removed
would bury it again. The directory's `Areas` call answers the roster and the
population together, which is also how an area that raised nothing is known
to exist, since a Me too and a raised item can only name areas that have
someone in them.

## Counting

For each board item that is not hidden (the set `ListBoardItems` answers):

1. People who raised: the distinct `raised_by` of publishable raised items
   grouped under it.
2. People who said Me too: the distinct `holder_id` of its endorsements.
3. A person in both counts once, as a raiser. So `raised_by + me_too` is the
   number of people behind the item, and the two columns never overlap.
4. Each person gets one area (above). Counts are per area.

A person's area is worked out once per tally for everyone: the directory's
answer first, else the stored `Area` of the earliest raised item of theirs with
one. So a person who raised under one board item and said Me too on another
sits in the same area on both. A name the directory places someone in that
`Areas` does not list is treated as a stored one: its own row, `population`
`null`. Asking for a board item that is hidden or does not exist answers an
empty list.

Two numbers already on the board read differently on purpose: `raised_by`
there counts raised items, not people, and its `me_too` includes raisers. The
operation's description says so.

## Operation and tool

| | |
| --- | --- |
| Operation id | `tally_board_items_by_area` |
| Route | `GET /board-items/tally-by-area`, optional query `board_item_id` to tally one |
| Action | `catalog.board_item.tally_by_area` |
| MCP tool | `forms_catalog_board_item_tally_by_area` (the host prefixes `forms_` and derives the name from the action) |
| Call | `TallyBoardItemsByArea(ctx, boardItemID string)` |
| Gate | staff read, never anonymous, not on `AnonymousActions` |

The route is a literal beside `/board-items/public` and
`/board-items/{key}/shared`, which the mux ranks above a wildcard. It adds
no wildcard of its own.

## Result

```json
[
  {
    "id": "...",
    "title": "...",
    "areas": [
      { "area": "North", "raised_by": 3, "me_too": 12, "population": 4100 },
      { "area": "East", "raised_by": 0, "me_too": 0, "population": 3800 },
      { "area": "", "raised_by": 0, "me_too": 2, "population": null }
    ]
  }
]
```

- Every area the directory lists is a row, zero if nobody there is behind the
  item. Rows follow the directory's order.
- An area a person's stored `Area` names that the directory does not list (the
  match ignores case, as `placeOf` does) is its own row with `population`
  `null`, after the listed ones.
- A person nobody can place goes in one row with `area` empty, last, and only
  when its counts are not both zero. It is named so the totals still add up to
  the people behind the item; it is never dropped.
- `population` is `null` on a row the directory does not list and on the row
  with no area. On a listed area it is a number: the wakala-api directory
  answers 0 where the actor module records no figure. The module does no
  division; the reader of both numbers does, as with scope figures today.
- A board item with nothing grouped and no Me too still appears, with the
  zero rows, as in `ListBoardItems`.

## Cost

Three reads for any number of items: the publishable raised items grouped
under the listed board items, their endorsements, and one `AreasOf` call over
the distinct person ids. Board items are bounded by the same cap
`ListBoardItems` uses.

## Left alone

- A person raising once as a register entry and again through an account counts
  twice: the ids differ, and the module does not read `UseLinkedEntries` here.
- The count is of now. A person who moves moves rows between areas in the next
  tally; no history is kept.
- Two areas with one name are one row. Area names are the key, as they already
  are on a raised item.

## Files that changed

Built, tested in `board_item_tally_test.go` and `routes/board_item_tally_test.go`
(catalog) and `internal/agencies/area_directory_test.go` (wakala-api). The action
is in the `board.read` permission (MUN1300), so a role other than the owner reads
the tally. Open: no screen or agent calls `forms_catalog_board_item_tally_by_area`. The list below is what
the design named.

Catalog (`mwanachama-backend-catalog`):

- `models/board_item.go`: `AreaTally`, `BoardItemAreaTally`, `AreaPopulation`.
- `board_item_tally.go` (new): `TallyBoardItemsByArea`.
- `catalog_manager.go`: `AreaDirectory`, `UseAreaDirectory`, the method on `CatalogManager`.
- `catalog.operations.json`: `tally_board_items_by_area`.
- Tests beside it, and the page this one points from: `documentation/2. design/raised-items.md`.

Hosts (the catalog is imported only by `mwanachama-wakala-api`, by `replace`
in `go.mod`; `mwanachama-backend-forms` and `mwanachama-backend-agency` do not import it):

- `mwanachama-wakala-api/internal/agencies/catalogs.go`: `buildCatalogManager` passes the directory after `NewCatalogManager`, from the instance's `ActorManager`. Nothing is passed today, in this repo, for `UseAreaResolver` either.
- `mwanachama-wakala-api/internal/agencies/area_directory.go` (new): the directory over the actor manager. A group is an area when its level has `is_area` set. `AreasOf` names the earliest such group a person is seated in (`ListActorsForGroup`), `Areas` lists those groups with `LatestGroupPopulation`.
- No edit in `mwanachama-wakala-api/cmd/server/catalog.go`,
  `internal/api/http/forms_catalog_routes.go`,
  `internal/api/http/mcp_tools_forms_catalog.go` or `mcp_tool_actions.go`: the
  route and the tool come out of `catalogroutes.OperatorRoutesFor` and
  `ToolsFor`, which read `catalog.operations.json`.
- `mwanachama-backend-agency/models/permissions_catalog.json`: the new read-only
  permission `board.read` holds `catalog.board_item.list` and
  `catalog.board_item.tally_by_area`, so a role other than the owner reads the
  tally without being able to group, hide or reassign. `board.manage` is
  unchanged.

Not changed, but diverged: `projects/hub-steward/code/wakala-api` carries its own
copy of the catalog (`modules/catalog/`), its own `permissions_catalog.json`
and MCP roster files under `internal/api/http/testdata/mcp_roster/`, and it is
the tree that wires `UseAreaResolver` today
(`internal/agencies/catalogs.go`, `ward_area.go`). It gets the same change only
when someone decides it should.
