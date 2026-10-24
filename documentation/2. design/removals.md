# Removals (CAT80)

A moderator taking something a resident raised off the public board is an act
with a reason, not a note written by convention. The municipality template
(`reports_catalog.json`, AGX14, member app decision 283) wanted that reason to
be required and to reach the person who raised the item; nothing enforced
either.

## What is built

- **A `removal` role**, declared in `catalog.blueprint.json` and carried by
  `models.Removal`: `raised_item_id`, `ground`, `removed_by`, `created_at`.
  `ground` and `removed_by` are required, and a ground of only whitespace is
  refused as well, so a removal always says why. A domain that removes nothing
  fills no object for the role and both calls answer `ErrRemovalsNotMade`. The
  agency example fills it as `removals`.
- **`RemoveRaisedItem(ctx, Removal)`** records the removal and, in the same
  transaction, makes the raised item unpublishable and takes it out of any
  board item, so it stops counting on the board and cannot be grouped again.
  The id and time are the store's; a body naming either is ignored. A raised
  item is removed at most once (`ErrRaisedItemAlreadyRemoved`), and an unknown
  one is `ErrRaisedItemNotFound`.
- **`ListRemovalsOfMine(ctx, raisedBy)`** answers the removals of the items
  `raisedBy` raised, with the ground and who removed them, and nothing of
  anybody else's.

`removed_by` and `raisedBy` are arguments here; a mount binds both from the
caller, never from a body, as the other raised-item writes do.

## Declared addresses (CAT132)

Member app decision 422 settled the MVP: staff remove a raised item with a
free-text ground, and the raiser reads why. Both calls are declared in
`catalog.operations.json`, neither is on `AnonymousActions`, and both are
pinned over the real route table in `routes/raised_item_removal_test.go`.

| Address | Action | Who is bound from the caller |
| --- | --- | --- |
| `POST /raised-items/{raisedItemID}/removal` | `catalog.raised_item.remove` | `removed_by`; the item is the path's, whatever the body names |
| `GET /raised-items/mine/removals` | `catalog.removal.list_mine` | the raiser |

The body of a removal carries only `ground`; an `id`, `created_at`,
`removed_by` or `raised_item_id` it claims is overwritten. A blank ground is
400 (`ErrInvalidRemoval`), an unknown item 404 (`ErrRaisedItemNotFound`), a
second removal of one item 409 (`ErrRaisedItemAlreadyRemoved`), and a mount
whose spec fills no `removal` object 404 (`ErrRemovalsNotMade`).

Who may remove is the mount's grant: the host gives
`catalog.raised_item.remove` to staff (owners, agents and holders of a staff
role, decision 411).

**Telling the raiser is the host's.** Decision 422 has the raiser told by an
Inbox line they cannot mute, as for their other item notices (decision 376).
That line is raised by wakala-api when its removal route succeeds (WK165);
this module records the removal and answers the raiser's read, and sends
nothing.

## Later, by decision 422

- Undoing a removal, and disputing one.
- A staff list of removals.
- A declared set of grounds; the ground stays free text for now.
- Hiding a whole board item stays Hide and Show again (decisions 373, 392);
  a removal applies to one raised item.
