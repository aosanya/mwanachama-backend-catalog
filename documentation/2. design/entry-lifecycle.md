# Entry lifecycle rules

Rules that the code states by name only. Each ties to the board row that
found it.

## A re-send never writes review state (CAT18, CAT35, CAT36, CAT42)

`UpsertEntry` and `UpdateEntry` share one rewrite. The reviewed content is the
name, the summary and the document, since an anonymous listing shows the card
and a keyed or public read shows the document.

- Unchanged reviewed content: the write leaves `visibility` and `approved_at`
  out of the row. A reviewer's `SetVisibility` that commits between the read
  and the write is therefore not overwritten, in either direction.
- Changed reviewed content: the entry returns to `pending`, `approved_at` is
  cleared and every live share key is retired. Retiring is what stops a key
  minted under an earlier private review from opening the entry the next
  time it is made private (CAT39).
- `UpdateEntry` ignores any `Visibility` or `ApprovedAt` the caller holds, so a
  stale snapshot cannot undo a decision.

## A share key issued during a visibility change (CAT17)

`IssueShareLink` checks the entry is private, inserts, then reads the entry
again. If the entry is no longer private every live key for it is retired and
the call answers `ErrNotPrivate`. The insert is not in a transaction, so the
re-read is what closes the window.

## A deleted entry frees its slug (CAT37)

The entry's slug uniqueness is the `slug_live` index, unique where
`deleted = false`, instead of a column constraint, so a deleted entry no
longer holds its slug.

Three things used to lean on the sterilised slug and are now explicit:

- `DeleteEntry` removes the entry's private notes, so a later entry at the same
  slug never reads them.
- `SeedEntries` skips a slug that any entry, deleted or not, ever held, so a
  deleted seed is not recreated on the next start.
- A database provisioned before this change still has the column constraint on
  `slug`; `spec.Migrate` does not drop it. Such a database frees the slug only
  once that constraint is dropped by hand.

## An anonymous view carries no writer (CAT40)

`OpenEntry` and `ListPublicEntries` return the entry with `submitted_by`
emptied. `GetEntry` and `ListEntries`, the operator's reads, keep it. On a
mount the field is the caller's internal subject, which a public reader has no
use for and which would let anyone correlate one operator's entries.

## Every refusal costs the store the same (CAT16)

`OpenEntry` looks the key up first whenever the caller sent one, then reads the
entry. A missing slug, a pending entry and a private entry opened with a wrong
key therefore each run the same two statements, so the clock does not tell a
stranger which private slugs exist. A caller sending no key runs one statement
in every case.
