# Services

CAT124, member app decisions 322, 324 and 326. A service is something an
agency offers that a member applies for and then follows: its name, what it
is, a fee shown as information (decision 288), the form it opens, and whether
it takes appointments.

## The call: an entry in a catalogue of its own, not a new object

The row left this to the builder: a catalogue kind (CAT49, decision 291) or
an object of its own. **A service is an entry in a catalogue mount kept for
services**, with its terms in the entry's document. Reasons:

- Everything the row asks for already exists on an entry. `name` is the
  name, `summary` is what it is, review state and visibility are publishing,
  `UpsertEntry` is how a template seeds the first ones, and `UpdateEntry` is
  editing. A new object would copy all of it.
- Who sees services is a catalogue's audience (CAT61, `WithAudience`), which
  works per mount. A services mount gets public, members or staff the same
  way any catalogue does, with no new check.
- CAT49's declared kinds are not built and their declaration site waits on
  the owner. Waiting on that would leave WK198 blocked; when kinds land, the
  terms below become the services kind's declared fields with no stored row
  changing, because they already live in the document.
- Decision 327 says catalogues read as records an agency publishes. A
  service is exactly that.

## The terms

`ServiceTerms` is the shape of a service entry's `doc`:

| Field | Meaning |
| --- | --- |
| `fee` | free text shown as information; nothing is charged here |
| `form` | the opaque id of the form Apply opens (forms owns it) |
| `appointments` | whether the service takes appointments (times are CAT124, which absorbed CAT125) |
| `opens` | `form` (the default) or `raise` |

`ServiceTermsOf(entry)` reads them and refuses, as `ErrInvalidServiceTerms`,
a document that is not an object, a `form` opening with no form, a `raise`
opening that names a form or takes appointments, and any other opening.
`ServiceEntry(slug, name, summary, terms)` builds the entry a staff edit or a
template seed writes.

## Checked on every write

The host wraps a services mount's manager in `WithServiceTerms(cm)`, the
same way it wraps one in `WithAudience`. The wrapped manager reads the terms
out of the document on every `CreateEntry`, `UpsertEntry` and `UpdateEntry`
and refuses a write whose terms do not hold with `ErrInvalidServiceTerms`
before anything is stored. The operations' error map answers that sentinel
400, so a staff edit or a template seed through the ordinary entry
operations gets a clean refusal. Nothing else on the manager changes:
visibility, deletion and reads pass straight through, and a mount not
wrapped stores any document, as before. Which mount is the services mount
stays the host's fact (WK198); the module never learns it.

**Raise something is one service** (decision 326): an entry whose terms say
`opens: raise`. The host opens the raise flow (CAT55) for it instead of a
form.

## Appointment times

Decision 325, CAT124. A time is **an object of its own, not a catalogue
kind**: two optional roles, `slot` and `booking`, declared in the blueprint
like `endorsement`. A domain that books no times fills neither, and every
method then answers `ErrTimesNotBooked`. A kind was ruled out because a time
has a capacity counted against live rows, which an entry's document cannot
hold or lock.

- `slot` is one time a service is offered: `subject_id` (the service entry,
  opaque here), `starts_at`, `ends_at` (RFC 3339, normalised to UTC, ending
  after it starts) and `capacity` (at least one; zero means one).
- `booking` is one holder's place at one time. It copies the time's
  `subject_id`, `starts_at` and calendar `day` when booked, so a holder's
  list and a day's list are one indexed read each. A cancellation is kept,
  not deleted: `cancelled_at`, `cancelled_by` and `cancel_reason`; a standing
  booking has an empty `cancelled_at`.

`Book(slot, holder)` runs in one transaction: read the time, refuse a second
standing booking by the same holder (`ErrAlreadyBooked`), count the standing
bookings and refuse when they reach the capacity (`ErrSlotFull`), then
insert. The holder is always the caller; an operation will bind it
`from: caller` and never read it from the body. Unlike the asset manager's
hold window, times are not checked for overlap: a service may offer two
times that overlap, each with its own capacity.

`CancelBooking(id, by, reason)` is the holder's own: another holder's
booking answers `ErrBookingNotFound`, exactly as a missing one.
`CancelBookingAsStaff(id, by, reason)` cancels anyone's and refuses an empty
reason (`ErrCancelReasonRequired`), so the holder always sees why. A second
cancel answers `ErrBookingCancelled`; the update is guarded on
`cancelled_at` being empty, so two racing cancels cannot both land. A
cancelled place frees the time.

`ListHolderBookings(holder)` lists a holder's bookings, cancelled included,
earliest first; `ListDayBookings(service, day)` lists one `YYYY-MM-DD` day's
bookings for staff, for one service or (empty service) every service.

## Not here yet

- Which mounts hold services is not declared yet: that needs CAT49's
  declared kinds or a per-mount setting, and which is the owner's call.
- No declared operations of their own: services are read and written through
  the existing entry operations on the services mount. Mounting it per agency
  is wakala-api's (WK198).
- Appointment times have no declared operations yet: the manager methods
  below are built and tested, and their addresses (book and cancel with the
  booker bound from the caller, the staff cancel and the day list gated
  apart) are what is left of CAT124.
- The template's first services are AGX18's data.
