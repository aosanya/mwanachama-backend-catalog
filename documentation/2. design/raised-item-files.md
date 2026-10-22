# A raised item's files (CAT126)

Member app decision 317 (2026-10-22): a raised item may carry files, each
seen by the raiser and by whoever answers for the host, and by the public
only once staff approve it for publishing on a publishable item. This takes
up CAT77's wish that a file be pinned to the item's own visibility instead of
sitting in a repository anyone can read.

## Records here, bytes elsewhere

The `raised_item_file` role holds one row per file attached to a raised item:
`raised_item_id`, `file_id`, `attached_by`, `attached_at`, and `approved_by`
/ `approved_at`, both empty until approved. `file_id` is an opaque string;
this module does not import `mwanachama-backend-files` and never reads the
bytes. A `(raised_item_id, file_id)` unique index keeps a file attached at
most once per item. A domain that takes no attachments fills no object for
the role, and every method then answers `ErrAttachmentsNotTaken`.

## The three operations

- `AttachFiles(raisedItemID, raiser, fileIDs)` — the raiser only: an item
  whose `raised_by` is not the caller answers `ErrRaisedItemNotFound`,
  exactly as a missing one. A file already attached is skipped, so a repeat
  is harmless. It answers the item's full list. A host attaches at raise by
  calling it straight after `RaiseItem`; `RaiseItem` itself carries no files.
- `ApproveFileForPublishing(raisedItemID, fileID, approvedBy)` — refused with
  `ErrRaisedItemNotPublishable` on an item that is not publishable, and
  `ErrAttachmentNotFound` for a file never attached. Approval is write-once:
  the update's `WHERE approved_at = ''` guard means a second approval, or two
  racing, leave the first approver and time in place.
- `ListRaisedItemFiles(raisedItemID)` — every file, approved or not. It is
  the raiser's and staff's read; who is staff is the host's to decide, the
  same as every other staff read here.

## The public read

`OpenSharedBoardItem` — the public board item and its shared link — now
carries `files`: the ids of approved files on the publishable raised items
grouped under it, in raised order, and an empty list when none is approved.
An unapproved file never appears there.

## Not built

No declared addresses: none of the raised-item operations are in
`catalog.operations.json` yet, so these are manager methods only, waiting on
the same decision as the rest. Withdrawing an approval and detaching a file
are not asked for by decision 317 and are not built.
