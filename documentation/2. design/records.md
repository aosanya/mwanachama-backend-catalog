# Records: one per holder per standing per subject (CAT48, CAT58)

A record is the latest set of values one holder sent about one subject,
with the time of every change kept beside it. The catalog does not know
what a subject or a holder is: a forms mount uses the form as the subject
and the person answering as the holder, but nothing here names either.

## Shape

`models.Record` carries `subject_id`, `holder_id`, `scope_id`, `doc` (a JSON object,
one key per field the host's subject asks about), `changes` (the history of
change times) and the two timestamps. The column is `doc` rather than
`values` because `values` is an SQL keyword and the table would not create.

The `record` role is keyed on `(subject_id, holder_id, scope_id)`, so there
is one row per holder per standing per subject by construction, with an
index on `subject_id` for the listing and the counts.

`scope_id` is the standing the holder sent under, opaque here. A forms mount
passes the group the member answered as (member app decision 247, amending
208): a trader who is also a teacher answers once as each and keeps two
records, each replaced only by a resend under its own standing. A host that
keeps one record per holder leaves it empty, and an empty standing is a key
like any other.

## Counted per standing and overall (CAT58, CAT128)

`catalog.record.list` and `catalog.record.count` take an optional
`scope_id` query parameter. Given, they read only the records sent under
that standing. Absent, they read every record on the subject. There is no
way to filter to the empty standing alone; a host that mixes empty and named
standings on one subject reads the empty ones from the full listing.

Every field's tally carries two counts. `counted` is the number of records
that carried a countable value, so each standing keeps the answer sent for
it (member app decision 247) and a holder who sent under two standings adds
two to the overall `counted`. `holders` is the number of distinct holders
behind those records, so the overall figure counts each person once (member
app decision 384, CAT128). Under one standing a holder has at most one
record, and the two counts agree.

The yes/no, option and number tallies stay per record: a holder who answered
differently under two standings shows both answers overall, because which
one would stand for the person is not decided.

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

## One record per holder per subject, opt-in (CAT66)

A host that needs one voice per holder on a subject, whatever standing they
send under, sets the setting `record.one_per_holder` to `true`
(`SettingOneRecordPerHolder`). `PutRecord` then refuses a write by a holder
who already has a record on that subject under a different standing, with
`ErrRecordHolderSpoke` (409). A resend under the same standing still replaces
the record, so the holder can change their answer but not add a second one.
The check runs in the write's transaction; the setting is read once before
it. Unset, records stay one per holder per standing per subject, as above.

The municipality template's consultation response maps here: the round is the
subject and the respondent the holder. The holder is whatever the host
passes; a host must bind it from the signed-in caller, never from a body,
since this module has no write route for records of its own.
