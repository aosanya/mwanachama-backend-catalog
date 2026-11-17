# Endorsements

An endorsement is one holder saying they share an item somebody already
raised: the member app's "me too" (member app decision 234, CAT55 here).

- **Optional role.** `endorsement` is declared in the blueprint, and a domain
  that counts none fills no object for it; `Endorse` and `CountEndorsements`
  then answer `ErrEndorsementsNotCounted`. The agency example fills it as
  `me_too`; the library example does not.
- **One per holder per item, by the key.** `(subject_id, holder_id)` is the
  primary key and `Endorse` inserts with `ON CONFLICT DO NOTHING`, so asking
  twice, or two requests racing, leaves one row. `Endorse` answers the item's
  count after the write, which is the "raised by N" a board shows.
- **Opaque ids.** The item and the holder are ids this module never reads for
  meaning. The holder must come from the host's caller, never a body.
- **Go-only for now.** No operation is declared yet: who may endorse what is
  the host's to wire, and the board item it is counted against is the rest of
  CAT55, not built in this slice.
- **Counted by area.** The count of people behind a board item, split by area
  against the area population, is the tally in [tally-by-ward.md](tally-by-ward.md)
  (hub issue MUN1283).
