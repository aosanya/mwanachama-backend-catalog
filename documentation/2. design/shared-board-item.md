# A board item's share link (CAT123)

Member app decision 311: any publishable item on the board answers a link
that opens without signing in. This page records the slice built here and
what is left.

## What is built

`OpenSharedBoardItem(ctx, key)` answers a `SharedBoardItem`:

| Field | Where it comes from |
| --- | --- |
| `title` | the board item's `title` |
| `theme` | the `topic` of the earliest publishable raised item grouped under it |
| `area` | the `area` of that same earliest raised item, as stored when it was raised (CAT81: the host's resolution from pin, then address, then the raiser's pick) |
| `me_too` | `CountEndorsements` on the board item, `0` where the domain counts none |
| `status` | the board item's `outcome`, empty until there is one |

The key is the board item's id. It is minted by the store as a random UUID,
so it is not guessable, and a board item has no other identity a link could
carry.

**Every miss is the same `ErrBoardItemNotFound`**: an empty key, an unknown
key, a board item with no publishable raised item grouped under it, and a
domain that keeps no board. A board item is publishable only while a
publishable raised item is grouped under it, so taking the last one off
closes its link. Splitting these would let anyone tell an unpublished board
item from one that never existed.

Nothing else is answered: who raised it, how many raised it, the raised
items themselves, their pins (rounded or not) and their typed addresses stay
off the link. `TestASharedBoardItemCarriesTheAreaButNeverTheAddressPinOrRaiser`
holds it by searching the encoded answer for the raiser, the address, the pin
and the raised item's id.

## What is left

- **Street.** The area is on the link since CAT81 landed. The street is not:
  a raised item's address is one typed line, and whether "the street" is a
  separate field or a part cut from that line is still the owner's (see
  [raised-items.md](raised-items.md), "Street versus typed address"). Until
  that is decided the link carries no street, the same way every other
  reader's view withholds the whole address.
- **An address (CAT123, 2026-10-23).** `GET /board-items/{key}/shared`,
  action `catalog.board_item.open_shared`, calls `OpenSharedBoardItem` and is
  on `AnonymousActions`, because decision 311 says the link opens without
  signing in. `ErrBoardItemNotFound` answers 404, so an unknown key and a
  board item with nothing publishable grouped under it read the same. Like
  every anonymous action it is not an MCP tool. A host with its own
  allowlist (`Mount.Anonymous`) names the action itself to publish it.
- **Me too through the link.** `Endorse` already records one per holder; that
  the holder is a signed-in member is the mount's to enforce, by binding the
  holder from the caller, never from the link.
- **A held Me too.** A Me too from a person whose join request is waiting has
  no natural place here: this module knows nothing of join requests, and
  `endorsement` has no state to hold one in. It needs either a `held` state on
  `endorsement` that `CountEndorsements` skips and an acceptance flips, or
  the host keeping the pending Me too and calling `Endorse` when the request
  is accepted. The second needs no change here.
