# Overdue raised items (CAT79)

A board that takes raised items promises them an answer within a clock. The
clock is a setting and every raised item carries the time it was raised, so
which ones are overdue is read straight off the store rather than assembled
by whoever remembers to ask.

## The read

`ListOverdueRaisedItems(ctx, at)` — declared as `GET /raised-items/overdue`,
action `catalog.raised_item.list_overdue`, never anonymous — answers the
raised items that are:

- raised at or before `at` less the answer clock, and
- still open: no reply has been sent on them, and they were not removed.

Oldest first. `at` is RFC 3339; empty means now. Passing it is what makes
the read testable against a stepped clock, and it lets a host age the queue
against the time its own sweep fired rather than the time the query ran.

"Open" is the module's own facts only: a reply or a removal closes an item.
Grouping an item under a board item, or a board item's outcome, does not —
those say what the board shows, not whether the person was answered.

## The clock

The setting `raised_item.answer_within_days` (exported as
`AnswerClockSetting`) holds a whole number of days of at least one. There is
no default: unset, the read answers `ErrSettingNotFound` (404), and a value
that is not such a number answers `ErrInvalidSetting` (400). A default would
be a policy the module chose for every domain; the municipality template
suggests ten, but marks it as the first number a council changes.

Days are calendar days. The template speaks of working days; counting them
needs a calendar of the domain's own (weekends, holidays), which this module
does not hold.

## What is not here

The schedule. The module runs nothing on its own; a host's scheduled job
calls this read and decides what to do with the list (notify, put it on a
board), exactly as `DisposeExpired` is called for retention.
