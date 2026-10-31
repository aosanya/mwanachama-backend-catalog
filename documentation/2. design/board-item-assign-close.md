# Assigning and closing a board item (CAT134, CAT135)

Member app decision 451 (owner, 2026-10-24), building on decisions 235, 272
and 389: a board item is assigned to a person or a role, whoever holds the
role picks it up, and only an explicit Close closes it.

## The fields

`board_item` carries four more fields:

- `assignee_kind` — `actor` (a person, named by the host's own id) or `role`
  (whose holders the host resolves), the same two kinds a task's assignee
  takes in taskmanager (decision 272). Empty means unassigned. It is declared
  as a string rather than an enum because an enum refuses the empty value an
  unassigned item holds; `models.IsAssigneeKind` checks the two kinds on
  assign.
- `assignee_id` — who, read as the kind says, opaque to this module.
- `closed_at` — when it was closed. Empty while open.
- `closed_by` — who closed it, the caller of Close. Empty while open.

All four are the store's: `PutOnBoard` clears whatever the body claimed, so
an item is always born open and unassigned.

## The operations

All three are gated; none is on `AnonymousActions`. A host puts them under
its Manage the board permission (agency AG89).

| Operation | Address | Action |
| --- | --- | --- |
| `AssignBoardItem` | `PUT /board-items/{boardItemID}/assignee` | `catalog.board_item.assign` |
| `UnassignBoardItem` | `DELETE /board-items/{boardItemID}/assignee` | `catalog.board_item.unassign` |
| `CloseBoardItem` | `POST /board-items/{boardItemID}/close` | `catalog.board_item.close` |

Assigning again replaces the assignee; unassigning leaves the item open.
Neither touches `closed_at`.

Close binds the closer from the mount's caller, never the body. It is
write-once: the update carries `closed_at = ''` in its own `WHERE`, so two
staff closing at once cannot both record themselves, and a second Close
answers `ErrBoardItemClosed` (409) without moving who or when.

## A closed item is final (CAT135)

Member app decision 467 (owner, 2026-10-25) extends 451: a closed board item
cannot be reopened, and assigning it is refused. The same problem raised
again starts a new item.

- There is no reopen operation, and none is to be added.
- `AssignBoardItem` writes through the same guarded update as Close, with
  `closed_at = ''` in its own `WHERE`, so an assign racing a Close cannot
  land on the closed item. A closed item answers `ErrBoardItemClosed` (409)
  and keeps its assignee, closer and close time; an unknown id still answers
  `ErrBoardItemNotFound`.
- Unassigning is not covered by the decision and is unchanged.

`TestAClosedBoardItemCannotBeAssigned` and, over the declared route table,
`TestAssigningAClosedBoardItemAnswersConflictThroughTheDeclaredTable` pin it.

## What never closes an item

A reply, a removal, an outcome and grouping all leave `closed_at` empty.
A reply or a removal takes a raised item off the overdue list
([overdue-raised-items.md](overdue-raised-items.md)), which is about whether
the raiser was answered, not whether the item is done.
`TestOnlyCloseClosesABoardItemAndOnlyOnce` pins it.
