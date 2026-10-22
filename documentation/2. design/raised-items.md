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
