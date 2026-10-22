# Raised items

CAT55, member app decisions 232 and 234. The optional `raised_item` role holds
something a holder raised for the mounting host to answer: a one-line `title`,
an opaque `topic`, whether it is `publishable`, and its raiser. A clerk filing
on the raiser's behalf sets `filed_by` and `channel` together; one without the
other is refused, so an item either was filed by its raiser or says who filed
it and how it arrived.

`RaiseItem` mints the id and the time, whatever the caller sent.
`ListRaisedItems` lists one raiser's items, oldest first. A domain filling no
object for the role builds and answers `ErrRaisedItemsNotTaken`.

Not here yet: where an item happened (whether place and ward are fields or
document paths waits on the owner), grouping publishable items into board
items, replies, the public outcome, and declared operations.

## Board items

The optional `board_item` role is one item on the public board: a `title` and
a public `outcome`, empty until there is one. It is kept only where raised
items are taken too; otherwise every board call answers `ErrBoardNotKept`.

A raised item is grouped under a board item by its own `board_item_id`, so an
item sits under at most one board item. `PutOnBoard` mints the id and time;
`GroupUnderBoardItem` refuses an item that is not `publishable`
(`ErrRaisedItemNotPublishable`), and answers `ErrRaisedItemNotFound` or
`ErrBoardItemNotFound` for an unknown id.

`ListBoardItems` lists board items oldest first, each with `raised_by`, the
number of publishable raised items grouped under it, and `me_too`, the
`endorsement` count with the board item's id as subject (zero where the domain
counts no endorsements). Visibility, place, replies and declared operations are
not here yet.

## Outcome and ungrouping

`SetBoardItemOutcome` sets a board item's public `outcome`; an empty string
clears it. `UngroupFromBoardItem` takes a raised item off whatever board item
it sat under by clearing its `board_item_id`, so it stops counting towards that
board item's `raised_by`. Both answer `ErrBoardItemNotFound` or
`ErrRaisedItemNotFound` for an unknown id and `ErrBoardNotKept` where no board
is kept. Neither is a declared operation yet.
