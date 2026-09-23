package models

// MaxSuggestionAttempts is how many times a build may fail before the
// suggestion stalls and waits for a person. Without a ceiling a proposal
// nothing can build would be retried by every run forever.
const MaxSuggestionAttempts = 10

// DefaultSuggestionPriority is where a newly filed proposal sorts. Lower is
// sooner, and the gap either side leaves room to reorder without renumbering
// the queue.
const DefaultSuggestionPriority = 100

// Suggestion is a proposal that something is worth adding to the catalog —
// an archetype worth mining, a title a patron wants bought. It is
// deliberately cheap: a title, a reason and a sketch, filed by whatever met
// the idea, and waiting on a person to approve it before anything is built.
//
// Nothing here names a domain. Anything domain-shaped — an agency's sector,
// a book's genre — belongs in Proposal, which is a free JSON document for
// exactly that reason.
type Suggestion struct {
	// ID is the storage key.
	ID string `json:"id"`

	// Slug is the slug the Entry would take, unique across suggestions and
	// checked against the catalog before filing.
	Slug string `json:"slug"`

	// Title is what the archetype would be called.
	Title string `json:"title"`

	// Why is the case for building it.
	Why string `json:"why,omitempty"`

	// Source is what filed it.
	Source SuggestionSource `json:"source"`

	// Status is where it has got to.
	Status SuggestionStatus `json:"status"`

	// Priority orders the queue, lowest first.
	Priority int `json:"priority"`

	// Attempts is how many builds have failed. Reset when an admin sends a
	// stalled suggestion back to approved.
	Attempts int `json:"attempts"`

	// Proposal is the sketch — goals, roles and objectives — as JSON text.
	Proposal string `json:"proposal,omitempty"`

	// RunID names the run that filed it.
	RunID string `json:"run_id,omitempty"`

	// CreatedAt is the RFC 3339 timestamp when it was filed.
	CreatedAt string `json:"created_at"`

	// DecidedAt is when it was approved or rejected, empty while proposed.
	DecidedAt string `json:"decided_at,omitempty"`
}

// SuggestionComment is one note on a suggestion — a run saying what is left
// after a failed attempt, or a person saying why they decided as they did.
type SuggestionComment struct {
	// ID is the storage key.
	ID string `json:"id"`

	// SuggestionID is the suggestion this is about.
	SuggestionID string `json:"suggestion_id"`

	// Author is which side wrote it.
	Author CommentAuthor `json:"author"`

	// RunID names the run, when a run wrote it.
	RunID string `json:"run_id,omitempty"`

	// Attempt is which build attempt this came out of. Zero for a comment
	// that is not about an attempt.
	Attempt int `json:"attempt,omitempty"`

	// Body is the note itself, never blank.
	Body string `json:"body"`

	// CreatedAt is the RFC 3339 timestamp when it was written.
	CreatedAt string `json:"created_at"`
}

// SuggestionMatch records that one Entry answers one Suggestion.
type SuggestionMatch struct {
	// SuggestionID is the suggestion that was met.
	SuggestionID string `json:"suggestion_id"`

	// Slug is the Entry that meets it.
	Slug string `json:"slug"`

	// RunID names the run that built it.
	RunID string `json:"run_id,omitempty"`

	// Note is what the run said about the match.
	Note string `json:"note,omitempty"`

	// MetAt is the RFC 3339 timestamp when the match was recorded.
	MetAt string `json:"met_at"`
}
