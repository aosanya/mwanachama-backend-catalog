# Raised items

CAT55, member app decisions 232 and 234. The optional `raised_item` role holds
something a holder raised for the mounting host to answer: a one-line `title`,
an opaque `topic`, whether it is `publishable`, and its raiser. A clerk filing
on the raiser's behalf sets `filed_by` and `channel` together; one without the
other is refused, so an item either was filed by its raiser or says who filed
it and how it arrived.

`RaiseItem` mints the id and the time, whatever the caller sent, and drops a
`filed_by` and `channel` the body claims: a member raising for themselves is
never a filing. `FileRaisedItem` is the clerk's write and keeps both, still
refusing one without the other.
`ListRaisedItems` lists one raiser's items, newest first (decision 421), as
the declared `catalog.raised_item.list_mine` says. A host that links a
signed-in caller to register entries passes `UseLinkedEntries`, a function
answering the entry ids linked to a caller; the raiser's own list then also
carries every item whose `subject_id` is one of them, so an item a clerk
filed for somebody shows in their list once their entry is linked. With no
function passed, or none linked, the list is the caller's own raises only.
The ids come from the host, never from the request. A domain filling no
object for the role builds and answers `ErrRaisedItemsNotTaken`.

## Anonymous items (hub issue MUN1402)

A raised item carries `anonymous`, false unless its raiser chose to stay
anonymous. The host still keeps who raised it. `AsSeenByOthers` leaves
`raised_by` empty for an anonymous item. `ListRaisedItemsOfPerson` is staff's
read of another person's items: it is `ListRaisedItems` for the named person,
leaving out every anonymous item unless the caller is the host owner
(`AsAgencyOwner`). `ListRaisedItemsOfSubject` leaves anonymous items out under
the same rule. The raiser's own `ListRaisedItems` keeps all of them. The
declared operation that would call `ListRaisedItemsOfPerson` is not in this
repository; the hub copy has it as `catalog.raised_item.list_of_person`.

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
back. Declared as `PUT /board-items/{boardItemID}/hidden` (`catalog.board_item.set_hidden`, body `{"hidden": bool}`, 204), gated: it is not on `AnonymousActions`.

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

**The street is the host's to resolve too** (decision 499). The item
carries a `street`, filled at `RaiseItem` by a `StreetResolver` the host
passes through `UseStreetResolver`: asked first with the typed address (the
street that address matches), then with the pin (the street at the pin's
address); if neither names one, the street the caller gave stands, and
without a resolver the given street stands. A resolver error refuses the
item, as for the area. The street is not redacted: `AsSeenByOthers` keeps
the area and the street and drops only the typed address and the exact pin,
so another reader learns which street but not which house.

**Who reads the exact pin.** `ListRaisedItems` is the raiser's own read and
answers the item whole. Every other read goes through
`RaisedItem.AsSeenByOthers`, which drops the typed address and rounds the
pin with `geo.RoundForPublic` (three decimals, about 100 m). Staff reads are
the host's: it reads the item whole and decides who is staff.

**Nearby.** `NearbyRaisedItems(topic, at, withinMetres)` lists published
items of one topic, where published means on the public board (decision 373):
publishable and grouped under a board item that is not hidden. An item never
grouped, ungrouped, or grouped under a hidden board item is left out, and a
mount that keeps no board answers an empty list. It lists the items whose
pin lies within the distance of `at` (great-circle, `geo.Distance`), nearest first, at most 200, each already seen-by-others.
Items with no pin are not near anything. The distance filter runs in Go over
the topic's published rows; a spatial index is not needed until a topic
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

**The place-setting write** is `RaiseItem` itself: the pin, address and
picked area arrive on `POST /raised-items` (or `POST /raised-items/filed`
for a clerk), both declared below.

**Still the owner's, and kept as built:**

- *A result cap.* The read keeps the manager's 200 nearest; whether the
  route should take a smaller limit or page is not decided.

## Declared addresses (CAT55)

Raising, filing, the board, Me too and replies are declared in
`catalog.operations.json`, every one gated:

| Address | Action | Who is bound from the caller |
| --- | --- | --- |
| `POST /raised-items` | `catalog.raised_item.raise` | `raised_by` |
| `POST /raised-items/filed` | `catalog.raised_item.file` | `filed_by`; `raised_by` is the register entry's id from the body (decision 374) |
| `GET /raised-items/mine` | `catalog.raised_item.list_mine` | the raiser |
| `POST /board-items`, `GET /board-items` | `catalog.board_item.put`, `.list` | — |
| `GET /board-items/public` | `catalog.board_item.list_public` | — |
| `GET /board-items/tally-by-area` | `catalog.board_item.tally_by_area` | — (see [tally-by-ward.md](tally-by-ward.md)) |
| `PUT`/`DELETE /raised-items/{raisedItemID}/board-item` | `catalog.raised_item.group`, `.ungroup` | — |
| `PUT /board-items/{boardItemID}/outcome` | `catalog.board_item.set_outcome` | — |
| `POST`/`GET /endorsements/{subject_id}` | `catalog.endorsement.add`, `.count` | the Me too holder |
| `POST`/`GET /raised-items/{raisedItemID}/replies` | `catalog.reply.send`, `.list` | the replier |
| `GET /raised-items/{raisedItemID}/replies/mine` | `catalog.reply.list_mine` | the reader |
| `POST /raised-items/{raisedItemID}/removal` | `catalog.raised_item.remove` | the remover (CAT132, see [removals.md](removals.md)) |
| `GET /raised-items/mine/removals` | `catalog.removal.list_mine` | the raiser |

Raising and filing are two actions so a mount grants filing on someone's
behalf to clerks alone. Listing replies with no raised item is refused.

`POST /raised-items` calls `RaiseItem`, which clears a `filed_by`,
`channel` and `subject_id` the body claims; `POST /raised-items/filed` calls `FileRaisedItem`.

The board has two reads (decision 373). `ListBoardItems` is the staff read:
every board item that is not hidden, including one with nothing grouped
under it, so staff can see a board item they have just started or emptied.
`ListPublicBoardItems` is the public read: the same list less every board
item with no publishable item grouped under it, so a board item emptied by
ungrouping or by a removal leaves the public board. Both are gated; whether
the public read is anonymous on a given mount is that mount's allowlist.

The public read answers one page at a time (decision 420), the way
`ListPublicEntries` caps its cards: `limit` and `offset` arrive as query
parameters, a page holds at most `MaxPublicBoardPage` (50) whatever the
caller asks for, zero or a negative limit means a full page, and a negative
offset starts at the first. The order is the staff read's, oldest first, so
a reader walking the pages sees each board item once while nothing changes.

Replies have two reads, split as the board's are (decision 411).
`catalog.reply.list` (`ListReplies`) is the staff read: any item's replies,
and who is staff is the mount's grant, never this module's answer.
`catalog.reply.list_mine` (`ListMyReplies`) binds the reader from the caller
and answers an item's replies only when the reader is its `raised_by` or its
`filed_by`; to anybody else, and to a caller the mount supplies as empty, the
item answers exactly as a missing one (`ErrRaisedItemNotFound`, 404), so the
read cannot be used to learn which items exist.

## A raiser the catalog holds (CAT75)

An item may be raised with no account and no contact details, so who raised
it is, at best, a contact string. Asking "who else has raised something this
year" or "is this area raising less than it used to" of `raised_by` asks about
arrivals, not people. `subject_id` is the optional answer: the id of an entry
in this same mount that the raiser is.

- Empty means the raiser is not in the catalog, and the item is tied to
  nobody. Every item raised before the field existed reads this way.
- Only a filing carries it (decision 421): `RaiseItem` clears a
  `subject_id` the body claims, as it clears `filed_by` and `channel`, so a
  resident cannot tie their raise to somebody else's register entry.
- When given on a filing, it must name an entry of this mount that
  is not deleted. Anything else is refused with `ErrEntryNotFound` (404) before
  a row is written, the same answer an unknown slug gets.
- `ListRaisedItemsOfSubject`, declared as `GET /raised-items/by-subject`
  (`catalog.raised_item.list_of_subject`), lists the items naming one entry,
  oldest first. It is gated like every other raised-item read and is not on
  `AnonymousActions`.
- The module checks only that the entry exists. Whether the caller may name
  that entry as themselves is the mount's to decide, and is not decided here.
