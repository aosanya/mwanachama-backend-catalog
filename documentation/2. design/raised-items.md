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

## Hiding a board item

A board item carries `hidden`, false when it is put on the board.
`SetBoardItemHidden` sets or clears it (member app decision 373), answering
`ErrBoardItemNotFound` for an unknown id, `ErrInvalidBoardItem` for an empty
one and `ErrBoardNotKept` where no board is kept. A hidden board item is left
out of `ListBoardItems`, and `OpenSharedBoardItem` answers it exactly as a
missing one, so hiding ends its share link; clearing the flag brings both
back. Not a declared operation yet.

## Replies

The optional `reply` role is one answer to a raised item: `raised_item_id`,
`body` and `replied_by`, all required. It is sent only where raised items are
taken too; otherwise both calls answer `ErrRepliesNotSent`.
`ReplyToRaisedItem` mints the id and time and refuses an unknown raised item
with `ErrRaisedItemNotFound`; `ListReplies` lists one raised item's replies by
`created_at` then id. Timestamps are kept to the second, so two replies in the
same second come back in id order rather than sending order. Who may read a
reply, and declared operations for it, are not here yet.

## A raised item's place (CAT81)

Member app decisions 305, 306 and 310. An item carries an optional `pin`
(a GeoJSON point, shared's S65 `geometry` type, nullable so "no pin" and a
pin at 0,0 differ), a typed `address`, and an `area`.

**The area is the host's to resolve.** This module knows no map of areas, so
the host passes an `AreaResolver` through `UseAreaResolver`. `RaiseItem`
asks it with the pin first; if that names no area, with the address; if that
names none either, the area the raiser picked stands, and with no pick the
item has no area. Without a resolver the picked area stands. A resolver error
refuses the item rather than storing it with a guessed area. The order lives
here so every host applies it the same way; what an area *is* stays opaque.

**Who reads the exact pin.** `ListRaisedItems` is the raiser's own read and
answers the item whole. Every other read goes through
`RaisedItem.AsSeenByOthers`, which drops the typed address and rounds the
pin with `geo.RoundForPublic` (three decimals, about 100 m). Staff reads are
the host's: it reads the item whole and decides who is staff.

**Nearby.** `NearbyRaisedItems(topic, at, withinMetres)` lists publishable
items of one topic whose pin lies within the distance of `at` (great-circle,
`geo.Distance`), nearest first, at most 200, each already seen-by-others.
Items with no pin are not near anything. The distance filter runs in Go over
the topic's publishable rows; a spatial index is not needed until a topic
holds more rows than that scan can afford.

**The nearby read is declared.** `GET /raised-items/nearby?topic=&at=&within=`,
action `catalog.raised_item.list_nearby`, calls `NearbyRaisedItems` through
the dispatch table; `at` is a JSON `[longitude, latitude]` pair and `within`
is metres. It binds no caller, because the answer is the same for everyone:
every item comes back seen-by-others, so the raiser's own item in a nearby
result is rounded too, and staff read exact pins through the host, not here.
It is not on `AnonymousActions`, so a mount gates it like any other read. A
missing topic, a distance of zero or less, or a point that is not a pair is
400 (`ErrInvalidRaisedItem` or the dispatcher's binding refusal); a mount
whose spec takes no raised items answers 404 (`ErrRaisedItemsNotTaken`).
Pinned over the real table in `routes/raised_item_nearby_test.go`.

**Not declared: the place-setting write.** The pin, address and picked area
arrive on `RaiseItem`, and raising is not a declared operation yet because
who `raised_by` is when a clerk files on someone's behalf is still the
owner's (CAT55). Declaring it here would settle that.

**Still the owner's, and kept as built:**

- *Street versus typed address.* The row says other readers get "the area
  and the street only", but the address is one typed line, so there is no
  street to keep apart from a house number; until that is decided the whole
  typed address is withheld from others.
- *What "published" means for nearby.* Today it is `publishable = true`;
  whether a moderator's approval or being on a board item should be needed
  instead is not decided.
- *A result cap.* The read keeps the manager's 200 nearest; whether the
  route should take a smaller limit or page is not decided.
