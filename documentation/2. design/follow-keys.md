# Follow keys (CAT76)

A raiser who gave no contact can follow the board item their arrival was
grouped under, because a board item is public anyway. Nothing opened their
own arrival and the replies written to them alone. A **follow key** does.

## What it is

- `POST /raised-items/{raisedItemID}/follow-key`
  (`catalog.raised_item.issue_follow_key`, gated) mints 32 random bytes,
  hex-encoded, and returns the raw key once. The store keeps only its SHA-256
  in `raised_item.follow_key_hash`, the same shape as `share_link.key_hash`.
  The field is `json:"-"`, so no response and no request body ever carries it.
- Issuing again overwrites the hash, which retires the previous key. One item
  has at most one live key; a rotation leaves no trail, unlike share links,
  because the only person it was handed to is the raiser.
- `GET /followed-items/{key}` (`catalog.raised_item.open_followed`) answers
  the raised item as the raiser filed it (address and exact pin included,
  since it is theirs) and every reply on it, oldest first.

## Why it is anonymous

Adding an anonymous action is a decision. This one is the whole point of the
row: the raiser has no account, so a gate in front of the read would leave
them exactly where they were. The key is the credential — 256 bits, matched by
hash, never stored raw — and it opens one item and nothing else.

## Every miss is the same

An empty key, an unknown key, a retired key and a catalog that takes no raised
items all answer `ErrRaisedItemNotFound` (404). The lookup is by hash alone,
so there is no id in the address to probe.

## Left for the owner

- **Who may issue a key** is the mount's grant on
  `catalog.raised_item.issue_follow_key`: the filing clerk at the counter, or
  the public filing flow handing it back at the moment of raising. The module
  does not mint one on raise, because that would change `RaiseItem`'s answer
  for every domain.
- The replies shown are every reply on the item; the blueprint has no "to the
  raiser alone" flag on `reply`, and every reply is already addressed to the
  raised item. If staff notes that the raiser must not see are wanted, that is
  a new field and a product decision.
