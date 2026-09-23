package mwanachamacatalog

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-catalog/spec"
)

// Entry, EntryPrivate, ShareLink and their enum types are aliases of their
// models. counterparts, so a caller needs only this package's import.
type (
	Entry             = models.Entry
	DocFilter         = models.DocFilter
	EntryPrivate      = models.EntryPrivate
	ShareLink         = models.ShareLink
	Suggestion        = models.Suggestion
	SuggestionComment = models.SuggestionComment
	SuggestionMatch   = models.SuggestionMatch
	MiningRun         = models.MiningRun
	Setting           = models.Setting
	PageView          = models.PageView
	Visibility        = models.Visibility
	LinkStatus        = models.LinkStatus
	SuggestionStatus  = models.SuggestionStatus
	SuggestionSource  = models.SuggestionSource
	CommentAuthor     = models.CommentAuthor
	PageKind          = models.PageKind
)

const (
	// VisibilityPending is the state every Entry lands in.
	VisibilityPending = models.VisibilityPending

	// VisibilityPublic is readable by anyone.
	VisibilityPublic = models.VisibilityPublic

	// VisibilityPrivate is readable only with a live share key.
	VisibilityPrivate = models.VisibilityPrivate

	// LinkStatusActive opens the Entry it belongs to.
	LinkStatusActive = models.LinkStatusActive

	// LinkStatusRetired no longer opens it.
	LinkStatusRetired = models.LinkStatusRetired

	// SuggestionStatusProposed is filed and waiting on a decision.
	SuggestionStatusProposed = models.SuggestionStatusProposed

	// SuggestionStatusApproved is cleared to be built.
	SuggestionStatusApproved = models.SuggestionStatusApproved

	// SuggestionStatusRejected will not be built.
	SuggestionStatusRejected = models.SuggestionStatusRejected

	// SuggestionStatusBuilt has an Entry answering it.
	SuggestionStatusBuilt = models.SuggestionStatusBuilt

	// SuggestionStatusStalled ran out of build attempts.
	SuggestionStatusStalled = models.SuggestionStatusStalled

	// SuggestionSourceRun is a mining run's own proposal.
	SuggestionSourceRun = models.SuggestionSourceRun

	// SuggestionSourceOwner is a person's.
	SuggestionSourceOwner = models.SuggestionSourceOwner

	// SuggestionSourceStray is met in passing rather than sought.
	SuggestionSourceStray = models.SuggestionSourceStray

	// CommentAuthorRun is a mining run.
	CommentAuthorRun = models.CommentAuthorRun

	// CommentAuthorAdmin is a person.
	CommentAuthorAdmin = models.CommentAuthorAdmin

	// PageKindIndex is the front page.
	PageKindIndex = models.PageKindIndex

	// PageKindEntry is one Entry's page.
	PageKindEntry = models.PageKindEntry

	// MaxSuggestionAttempts is how many builds may fail before a suggestion
	// stalls.
	MaxSuggestionAttempts = models.MaxSuggestionAttempts
)

// ListFilter narrows [CatalogManager.ListEntries]. A zero filter lists
// every non-deleted Entry.
type ListFilter struct {
	// Visibility, when set, keeps only Entries in that state.
	Visibility Visibility

	// Docs keeps only Entries whose document holds the given value at the
	// given path. This is how a domain filters on its own vocabulary —
	// {Path: "_template.sector", Value: "health"} for an agency catalog,
	// {Path: "subject", Value: "history"} for a library — without this
	// package learning either word.
	Docs []DocFilter

	// Search, when set, keeps Entries whose name or summary contains it,
	// case-insensitively.
	Search string

	// Limit caps the rows returned. Zero means no cap.
	Limit int
}

// CatalogManager is the single interface for the catalog: the documents,
// the operator's private notes on each, and the keys that open a private
// one. It names no domain — see this package's doc.go.
type CatalogManager interface {
	// CreateEntry stores a new Entry as pending, whatever Visibility it
	// carries, and returns it with its generated id and timestamps.
	CreateEntry(ctx context.Context, e Entry) (Entry, error)

	// UpsertEntry creates the Entry or replaces the document and card of an
	// existing one. Replacing the document of an approved Entry returns it
	// to pending, so nothing unreviewed is ever readable.
	UpsertEntry(ctx context.Context, e Entry) (Entry, error)

	// GetEntry reads one Entry regardless of Visibility. This is the
	// operator's read; OpenEntry is the reader's.
	GetEntry(ctx context.Context, slug string) (Entry, error)

	// OpenEntry reads one Entry as a public reader. A public Entry
	// opens with any key. A private one opens only with a live share key. A
	// pending Entry, a missing slug and a wrong key are all
	// ErrEntryNotFound.
	OpenEntry(ctx context.Context, slug, key string) (Entry, error)

	// UpdateEntry replaces the mutable fields of an existing Entry.
	UpdateEntry(ctx context.Context, e Entry) (Entry, error)

	// SetVisibility moves an Entry between pending, public and private,
	// stamping ApprovedAt when it leaves pending and clearing it on the way
	// back.
	SetVisibility(ctx context.Context, slug string, v Visibility) (Entry, error)

	// DeleteEntry soft-deletes an Entry.
	DeleteEntry(ctx context.Context, slug string) error

	// ListEntries returns the cards matching f, ordered by name, never
	// carrying Doc.
	ListEntries(ctx context.Context, f ListFilter) ([]Entry, error)

	// PutPrivate writes the private parts of one Entry. Empty fields leave
	// the stored value alone, so a caller may update the rank without
	// resending the readme.
	PutPrivate(ctx context.Context, p EntryPrivate) (EntryPrivate, error)

	// GetPrivate reads the private parts of one Entry.
	GetPrivate(ctx context.Context, slug string) (EntryPrivate, error)

	// IssueShareLink mints a key for a private Entry, returning the link
	// and the raw key. The raw key is returned only here and never stored.
	IssueShareLink(ctx context.Context, slug, label, by string) (ShareLink, string, error)

	// RetireShareLink stops a link opening its Entry, keeping the row.
	RetireShareLink(ctx context.Context, id string) (ShareLink, error)

	// ListShareLinks returns every link issued for one Entry, newest first.
	ListShareLinks(ctx context.Context, slug string) ([]ShareLink, error)

	// FileSuggestion proposes something worth adding. It always lands
	// proposed, and refuses a slug that is already an Entry or already
	// proposed.
	FileSuggestion(ctx context.Context, s Suggestion) (Suggestion, error)

	// GetSuggestion reads one proposal.
	GetSuggestion(ctx context.Context, id string) (Suggestion, error)

	// ListSuggestions returns proposals in the given statuses, best
	// priority first. No statuses means all of them.
	ListSuggestions(ctx context.Context, statuses ...SuggestionStatus) ([]Suggestion, error)

	// DecideSuggestion moves a proposal between proposed, approved and
	// rejected, optionally reordering it. A non-zero priority replaces the
	// stored one. It cannot set "built".
	DecideSuggestion(ctx context.Context, id string, status SuggestionStatus, priority int) (Suggestion, error)

	// RecordSuggestionAttempt counts a build that did not finish and keeps
	// the reason as a comment. At MaxSuggestionAttempts the proposal
	// stalls.
	RecordSuggestionAttempt(ctx context.Context, id, runID, comment string) (Suggestion, error)

	// CommentOnSuggestion adds a note that is not about a build attempt.
	CommentOnSuggestion(ctx context.Context, id string, author CommentAuthor, body string) (SuggestionComment, error)

	// ListSuggestionComments returns every note on a proposal, oldest
	// first.
	ListSuggestionComments(ctx context.Context, id string) ([]SuggestionComment, error)

	// FulfilSuggestion records that an Entry answers a proposal and marks
	// it built.
	FulfilSuggestion(ctx context.Context, id, slug, runID, note string) (Suggestion, error)

	// ListSuggestionMatches returns the Entries answering a proposal.
	ListSuggestionMatches(ctx context.Context, id string) ([]SuggestionMatch, error)

	// RecordRun stores one mining run's report.
	RecordRun(ctx context.Context, r MiningRun) (MiningRun, error)

	// ListRuns returns run reports, newest first.
	ListRuns(ctx context.Context, limit int) ([]MiningRun, error)

	// PutSetting writes one run-time knob.
	PutSetting(ctx context.Context, key, value string) (Setting, error)

	// GetSetting reads one.
	GetSetting(ctx context.Context, key string) (Setting, error)

	// RecordPageView adds one to a page's running total. An agency view
	// counts only against a slug that exists.
	RecordPageView(ctx context.Context, page PageKind, slug string) error

	// ListPageViews returns every counted page, most viewed first.
	ListPageViews(ctx context.Context) ([]PageView, error)
}

type catalogManager struct {
	db  *gorm.DB
	st  *store
	now func() string
}

// NewCatalogManager builds a CatalogManager over db, reading and writing the
// tables s declares. spec.Migrate must have run for the same db and s.
//
// The spec is checked against the types this package carries before anything
// is served: a declared column with no field to hold it, or a field with no
// column to land in, fails here rather than dropping a value on every write.
func NewCatalogManager(db *gorm.DB, s *spec.Spec) (CatalogManager, error) {
	st, err := newStore(db, s, map[string]any{
		roleEntry:             models.Entry{},
		roleEntryPrivate:      models.EntryPrivate{},
		roleShareLink:         models.ShareLink{},
		roleSuggestion:        models.Suggestion{},
		roleSuggestionComment: models.SuggestionComment{},
		roleSuggestionMatch:   models.SuggestionMatch{},
		roleRun:               models.MiningRun{},
		roleSetting:           models.Setting{},
		rolePageView:          models.PageView{},
	})
	if err != nil {
		return nil, fmt.Errorf("NewCatalogManager: %w", err)
	}
	return &catalogManager{db: db, st: st, now: nowRFC3339}, nil
}

// q starts a query against the table playing one role.
func (m *catalogManager) q(ctx context.Context, role string) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.st.table(role))
}

// take reads at most one row into out, answering notFound when there is
// none. Rows come back as maps rather than structs, because the columns are
// the spec's to declare and this package no longer restates them.
func (m *catalogManager) take(q *gorm.DB, role string, out any, notFound error) error {
	var rows []map[string]any
	if err := q.Limit(1).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return notFound
	}
	return decode(m.st.object(role), rows[0], out)
}

// listOf reads every row q matches. A free function rather than a method
// because Go has no generic methods, and the alternative is the same decode
// loop written out once per listing.
func listOf[T any](m *catalogManager, q *gorm.DB, role string) ([]T, error) {
	var rows []map[string]any
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	o := m.st.object(role)
	out := make([]T, 0, len(rows))
	for _, r := range rows {
		var v T
		if err := decode(o, r, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// insert writes v as a new row.
func (m *catalogManager) insert(ctx context.Context, role string, v any) error {
	row, err := encode(m.st.object(role), v)
	if err != nil {
		return err
	}
	return m.q(ctx, role).Create(row).Error
}
