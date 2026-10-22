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

Nothing else is answered: who raised it, how many raised it, and the raised
items themselves stay off the link.

## What is left

- **Area and street.** A raised item has no place until CAT81 lands; the
  link gains them then.
- **An address.** The method is on the manager only. Board items have no
  operation in `catalog.operations.json` yet, so no route serves this, and
  nothing is added to `AnonymousActions`. Declaring the anonymous read is the
  deliberate step that publishes it, and is the host's (wakala-api WK196) or a
  follow-up here.
- **Me too through the link.** `Endorse` already records one per holder; that
  the holder is a signed-in member is the mount's to enforce, by binding the
  holder from the caller, never from the link.
- **A held Me too.** A Me too from a person whose join request is waiting has
  no natural place here: this module knows nothing of join requests, and
  `endorsement` has no state to hold one in. It needs either a `held` state on
  `endorsement` that `CountEndorsements` skips and an acceptance flips, or
  the host keeping the pending Me too and calling `Endorse` when the request
  is accepted. The second needs no change here.
