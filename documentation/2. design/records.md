# Records: one per holder per subject (CAT48, first slice)

A record is the latest set of values one holder sent about one subject,
with the time of every change kept beside it. The catalog does not know
what a subject or a holder is: a forms mount uses the form as the subject
and the person answering as the holder, but nothing here names either.

## Shape

`models.Record` carries `subject_id`, `holder_id`, `values` (a JSON object,
one key per field the host's subject asks about), `changes` (the history of
change times) and the two timestamps.

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

## Not in this slice

The `record` role in the blueprint and both example specs, the upsert the
host calls on each send, the per-subject list and count operations in
`catalog.operations.json`, and the read gate (the host's authorizer: the
agency's owners and agents and the subject's group leaders, never a holder).
