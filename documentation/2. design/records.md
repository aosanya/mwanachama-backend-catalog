# Records: one per holder per subject (CAT48)

A record is the latest set of values one holder sent about one subject,
with the time of every change kept beside it. The catalog does not know
what a subject or a holder is: a forms mount uses the form as the subject
and the person answering as the holder, but nothing here names either.

## Shape

`models.Record` carries `subject_id`, `holder_id`, `doc` (a JSON object,
one key per field the host's subject asks about), `changes` (the history of
change times) and the two timestamps. The column is `doc` rather than
`values` because `values` is an SQL keyword and the table would not create.

The `record` role is keyed on `(subject_id, holder_id)`, so there is one row
per holder per subject by construction, with an index on `subject_id` for
the listing and the counts.

## Counts are read off the value's own JSON shape

`TallyRecords` works counts out from a set of records without the catalog
learning the host's field kinds:

| Value | Counted as |
| --- | --- |
| `true` / `false` | a yes/no tally |
| a string | one option tally |
| an array of strings | one tally per option chosen |
| a number (an amount or a currency value alike) | average, lowest and highest |
| `null` or absent | not counted |

`counted` is the number of records that carried a countable value for that
field, so a skipped field does not dilute an average or a share.

## A domain that keeps no records

The role is **optional**, unlike every other role here. `NewCatalogManager`
adds the `record` carrier only when the spec fills the role, so a domain
whose spec says nothing about records (wakala-api's `wakala.catalog` and
`wakala.reports` declarations today) still builds, and every record call
answers `ErrRecordsNotKept` (404). Both shipped examples fill it.

## The upsert is the host's, not an address

`PutRecord` is a Go call the host makes on each send, never a declared
operation: no caller may write a record over the wire, so a member cannot
write somebody else's, or their own outside a send. It replaces `doc` whole,
keeps the first send's `id` and `created_at`, moves `last_updated`, and
appends the send's time to `changes`. A caller-supplied `id` or `changes` is
ignored. Insert-or-update runs in one transaction; two first sends racing
on one pair can still meet the primary key, which the host sees as an error
rather than a lost write.

## Reads are declared, and gated by the mount

`catalog.record.list` (`GET /records/{subject_id}`) and
`catalog.record.count` (`GET /records/{subject_id}/counts`). Neither is in
`AnonymousActions`. Who may call them is the mount's authorizer: the
agency's owners and agents and the subject's group leaders, never a holder
(member app decisions 208, 209). The catalog itself cannot tell a leader
from a member, so the grant is wakala-api's to make when it mounts records
(WK153).
