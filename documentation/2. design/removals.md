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

## Not declared

Neither call is a declared operation. Raising and replying are not declared
either, because who may call them is still the owner's question (CAT55);
declaring removal would settle the same question for it.

## Open for the owner

- Can a removal be undone, and if so is the undoing an act with its own
  ground?
- Can the raiser dispute a removal, and who decides the dispute?
- Should the ground be free text, or one of a declared set of reasons with
  free text beside it?
- Who besides the raiser reads removals: every holder of the moderation power,
  or only the remover and those above them? There is no staff read yet.
- Does a removal also apply to a board item as a whole, or only to what one
  resident raised?
- Should the raiser be notified (through comm) when a removal is recorded?
