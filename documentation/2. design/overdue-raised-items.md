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

"Still open" here means still waiting for an answer, and it is the module's
own facts only: a reply or a removal takes an item off this list. Neither
*closes* anything. A board item closes only by an explicit Close from staff
(decision 451, CAT134 — see [board-item-assign-close.md](board-item-assign-close.md)),
and an item answered or removed stays on the board, open, until then
(decision 389). A reply is always to the raiser — it is sent on the raised
item and is what the raiser reads — so "only a reply to the raiser answers an
item" (decision 390) is exactly this rule. Grouping an item under a board
item, or a board item's outcome, does not answer it — those say what the
board shows, not whether the person was answered. `TestOnlyAReplyToTheRaiserTakesAnItemOffTheOverdueList`
pins it: an item grouped under a board item with an outcome set is still on
the list, and drops off only once a reply is sent.

## The clock

The setting `raised_item.answer_within_days` (exported as
`AnswerClockSetting`) holds a whole number of days of at least one. Unset, it
defaults to ten (`DefaultAnswerClock`), the template's answer clock (decision
389); a value that is not such a number answers `ErrInvalidSetting` (400).

Days are weekdays, Monday to Friday, with no holidays (decision 390): an item
is overdue once `at`, counted back that many weekdays, falls at or after the
moment it was raised. Counting back steps one calendar day at a time and
counts only the steps that land on a weekday, so a clock read on a Saturday
or Sunday counts from the Friday before. Holidays are not counted because
they are a calendar of the domain's own, which this module does not hold.

## The acknowledgement clock

Not built. Decision 233 asks for both clocks to be listed — the five-weekday
acknowledgement clock beside the ten-weekday answer clock — but a raised item
carries no acknowledged date and there is no Acknowledge operation to stamp
one. Both are CAT56's to add; a second list keyed on the acknowledged date
follows once they exist, counted in the same weekdays.

## What is not here

The schedule. The module runs nothing on its own; a host's scheduled job
calls this read and decides what to do with the list (notify, put it on a
board), exactly as `DisposeExpired` is called for retention.
