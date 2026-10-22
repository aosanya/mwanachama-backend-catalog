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
| `appointments` | whether the service takes appointments (times are CAT125) |
| `opens` | `form` (the default) or `raise` |

`ServiceTermsOf(entry)` reads them and refuses, as `ErrInvalidServiceTerms`,
a document that is not an object, a `form` opening with no form, a `raise`
opening that names a form or takes appointments, and any other opening.
`ServiceEntry(slug, name, summary, terms)` builds the entry a staff edit or a
template seed writes.

**Raise something is one service** (decision 326): an entry whose terms say
`opens: raise`. The host opens the raise flow (CAT55) for it instead of a
form.

## Not here yet

- The terms are checked when read and when built through `ServiceEntry`, not
  on every entry write: an entry write does not know its mount holds
  services. Checking on write needs either CAT49's declared kinds or a
  per-mount setting, and which is the owner's call.
- No declared operations of their own: services are read and written through
  the existing entry operations on the services mount. Mounting it per agency
  is wakala-api's (WK198).
- Appointment times and capacity are CAT125.
- The template's first services are AGX18's data.
