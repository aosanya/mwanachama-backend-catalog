package models

// Closed vocabularies for the catalog. As with
// mwanachama-backend-accounting's vocabulary.go, none of these becomes a
// database CHECK: a value added without an arm here should fail at the
// write, not relocate the same failure into a migration nobody re-reads.

// Visibility is who may read an Entry's document.
type Visibility string

const (
	// VisibilityPending is the state every Entry lands in. Nothing outside
	// the review surface may read it.
	VisibilityPending Visibility = "pending"

	// VisibilityPublic is readable by anyone, with no token.
	VisibilityPublic Visibility = "public"

	// VisibilityPrivate is readable only by presenting a live ShareLink key.
	VisibilityPrivate Visibility = "private"
)

var visibilities = map[Visibility]bool{
	VisibilityPending: true, VisibilityPublic: true, VisibilityPrivate: true,
}

// IsVisibility reports whether v is one of the three states.
func IsVisibility(v Visibility) bool { return visibilities[v] }

// LinkStatus is whether a ShareLink still opens the Entry it belongs to.
type LinkStatus string

const (
	// LinkStatusActive opens the Entry.
	LinkStatusActive LinkStatus = "active"

	// LinkStatusRetired no longer opens it. Retired links are kept rather
	// than deleted so a rotation leaves a trail of what was handed out and
	// when it stopped working.
	LinkStatusRetired LinkStatus = "retired"
)

var linkStatuses = map[LinkStatus]bool{
	LinkStatusActive: true, LinkStatusRetired: true,
}

// IsLinkStatus reports whether s is one of the two states.
func IsLinkStatus(s LinkStatus) bool { return linkStatuses[s] }

// SuggestionStatus is where a proposal has got to.
type SuggestionStatus string

const (
	// SuggestionStatusProposed is filed and waiting on a decision.
	SuggestionStatusProposed SuggestionStatus = "proposed"

	// SuggestionStatusApproved is cleared to be built.
	SuggestionStatusApproved SuggestionStatus = "approved"

	// SuggestionStatusRejected will not be built.
	SuggestionStatusRejected SuggestionStatus = "rejected"

	// SuggestionStatusBuilt has an Entry answering it.
	SuggestionStatusBuilt SuggestionStatus = "built"

	// SuggestionStatusStalled ran out of build attempts and waits for an
	// admin to send it back to approved.
	SuggestionStatusStalled SuggestionStatus = "stalled"
)

var suggestionStatuses = map[SuggestionStatus]bool{
	SuggestionStatusProposed: true, SuggestionStatusApproved: true,
	SuggestionStatusRejected: true, SuggestionStatusBuilt: true,
	SuggestionStatusStalled: true,
}

// IsSuggestionStatus reports whether s is one of the five states.
func IsSuggestionStatus(s SuggestionStatus) bool { return suggestionStatuses[s] }

// SuggestionSource is what filed a proposal.
type SuggestionSource string

const (
	// SuggestionSourceRun is a mining run's own proposal.
	SuggestionSourceRun SuggestionSource = "run"

	// SuggestionSourceOwner is a person's. A run may not file one.
	SuggestionSourceOwner SuggestionSource = "owner"

	// SuggestionSourceStray is something a run met in passing rather than
	// went looking for.
	SuggestionSourceStray SuggestionSource = "stray"
)

var suggestionSources = map[SuggestionSource]bool{
	SuggestionSourceRun: true, SuggestionSourceOwner: true, SuggestionSourceStray: true,
}

// IsSuggestionSource reports whether s is one of the three.
func IsSuggestionSource(s SuggestionSource) bool { return suggestionSources[s] }

// CommentAuthor is which side wrote a comment on a suggestion.
type CommentAuthor string

const (
	// CommentAuthorRun is a mining run saying what is left after an attempt.
	CommentAuthorRun CommentAuthor = "run"

	// CommentAuthorAdmin is a person, e.g. saying why they rejected one.
	CommentAuthorAdmin CommentAuthor = "admin"
)

var commentAuthors = map[CommentAuthor]bool{
	CommentAuthorRun: true, CommentAuthorAdmin: true,
}

// IsCommentAuthor reports whether a is one of the two.
func IsCommentAuthor(a CommentAuthor) bool { return commentAuthors[a] }

// PageKind is which page a view was counted against. Two kinds, named for
// their shape rather than their subject: a storefront's front page and a
// library's front page are both PageKindIndex.
type PageKind string

const (
	// PageKindIndex is the front page, which has no slug.
	PageKindIndex PageKind = "index"

	// PageKindEntry is one Entry's page, counted against its slug.
	PageKindEntry PageKind = "entry"
)

var pageKinds = map[PageKind]bool{
	PageKindIndex: true, PageKindEntry: true,
}

// IsPageKind reports whether p is one of the two.
func IsPageKind(p PageKind) bool { return pageKinds[p] }
