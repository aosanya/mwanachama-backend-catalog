# Who sees a catalogue, and what stays with staff (CAT61)

Member app decisions 286 and 288, 2026-10-22: each catalogue chooses who sees
it (anyone without signing in, members only, or staff only), and an entry's
cost and margin are only ever seen by staff.

## The audience is the mount's

`WithAudience(cm, audience, standingOf)` wraps a `CatalogManager` for one
mount. The audience is one of `public`, `members` or `staff`; anything else is
refused when the wrapper is built, so a typo cannot open a catalogue.

The caller's standing — `StandingNone`, `StandingMember` or `StandingStaff` —
is a fact the host supplies through `standingOf(ctx)`, the same way comm's
`UseBelonging` and forms' `ReachedOnly` take the host's word for who somebody
is. This module never learns what makes somebody a member or staff. A nil
`standingOf` gives every caller `StandingNone`, so a members or staff
catalogue mounted without one admits nobody.

The wrapper is a decorator rather than a constructor option, so a mount
chooses its audience without the manager's constructor or `SpecFor` changing.

## Every read checks it

`OpenEntry`, `GetEntry`, `ListPublicEntries` and `ListEntries` all ask
whether the caller's standing reaches the audience first. A caller who falls
short gets exactly what a missing entry gets: `ErrEntryNotFound` (404) for a
single read and an empty list for a listing — never a distinct refusal, so a
members catalogue cannot be told apart from an empty one by somebody signed
out.

## Below staff, only published entries (CAT129)

Member app decision 395, 2026-10-24: a reader the audience admits but who is
not staff sees only published entries. An entry still `pending` review
answers `GetEntry` and `OpenEntry` with `ErrEntryNotFound`, exactly as a
missing slug does, and is left out of `ListEntries`; a listing asking for
`visibility=pending` comes back empty. Staff keep every entry, pending
included. The audience check runs first, so a signed-out reader of a members
catalogue still gets not found for every entry, published or not (CAT61).

`ListEntries` below staff asks the manager with no limit, drops the pending
entries, then applies the caller's limit, so a limit counts published entries
only and a pending one never takes a published one's place.
`TestBelowStaffAPendingEntryAnswersExactlyAsAMissingSlug` was mutation-checked
by disabling the pending check.

Writes are not fenced by the audience; they are gated by the mount's
authorizer as before.

## Prices: public fields and private parts

The module names no domain, so it has no `price`, `cost` or `margin` field.
A domain puts the selling price in the entry's document (`entry.doc`), which
every reader who reaches the catalogue may see, and cost and margin in the
entry's private parts (`entry_private.doc`, written through `PutPrivate`).

Private parts were already served only by `GetPrivate`, an operator address
no anonymous mount opens. Under `WithAudience` they are also refused to any
caller below `StandingStaff`, read or write, as `ErrEntryNotFound` — whatever
the catalogue's audience. `OpenEntry` and the listings never carry them,
since they read the entry alone; `TestPrivatePartsNeverReachANonStaffRead`
pins both.

## Not built here

- Storing the audience in the catalogue's settings so a studio can change it
  (wakala-api WK188 reads it and builds the wrapper per mount).
- What a staff standing means in a given agency, which is the host's.
