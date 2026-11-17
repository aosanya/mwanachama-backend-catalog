package mwanachamacatalog

import (
	"context"
	"fmt"
	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-shared/geo"
	"github.com/aosanya/mwanachama-backend-shared/spec"
	"gorm.io/gorm"
	"time"
)

type (
	Entry                  = models.Entry
	DocFilter              = models.DocFilter
	EntryPrivate           = models.EntryPrivate
	ShareLink              = models.ShareLink
	Suggestion             = models.Suggestion
	SuggestionComment      = models.SuggestionComment
	SuggestionMatch        = models.SuggestionMatch
	MiningRun              = models.MiningRun
	Setting                = models.Setting
	PageView               = models.PageView
	Visibility             = models.Visibility
	LinkStatus             = models.LinkStatus
	SuggestionStatus       = models.SuggestionStatus
	SuggestionSource       = models.SuggestionSource
	CommentAuthor          = models.CommentAuthor
	PageKind               = models.PageKind
	Record                 = models.Record
	Endorsement            = models.Endorsement
	Slot                   = models.Slot
	Booking                = models.Booking
	RaisedItem             = models.RaisedItem
	BoardItem              = models.BoardItem
	BoardItemTally         = models.BoardItemTally
	AreaTally              = models.AreaTally
	BoardItemAreaTally     = models.BoardItemAreaTally
	AreaPopulation         = models.AreaPopulation
	SharedBoardItem        = models.SharedBoardItem
	Reply                  = models.Reply
	FollowKey              = models.FollowKey
	FollowedRaisedItem     = models.FollowedRaisedItem
	FollowedRaisedItemView = models.FollowedRaisedItemView
	MyRaisedItem           = models.MyRaisedItem
	Removal                = models.Removal
	RaisedItemFile         = models.RaisedItemFile
	FieldTally             = models.FieldTally
	ScopeFigure            = models.ScopeFigure
)

const (
	VisibilityPending        = models.VisibilityPending
	VisibilityPublic         = models.VisibilityPublic
	VisibilityPrivate        = models.VisibilityPrivate
	LinkStatusActive         = models.LinkStatusActive
	LinkStatusRetired        = models.LinkStatusRetired
	SuggestionStatusProposed = models.SuggestionStatusProposed
	SuggestionStatusApproved = models.SuggestionStatusApproved
	SuggestionStatusRejected = models.SuggestionStatusRejected
	SuggestionStatusBuilt    = models.SuggestionStatusBuilt
	SuggestionStatusStalled  = models.SuggestionStatusStalled
	SuggestionSourceRun      = models.SuggestionSourceRun
	SuggestionSourceOwner    = models.SuggestionSourceOwner
	SuggestionSourceStray    = models.SuggestionSourceStray
	CommentAuthorRun         = models.CommentAuthorRun
	CommentAuthorAdmin       = models.CommentAuthorAdmin
	PageKindIndex            = models.PageKindIndex
	PageKindEntry            = models.PageKindEntry
	MaxSuggestionAttempts    = models.MaxSuggestionAttempts
)

type PublishSchedule struct {
	Slugs     []string `json:"slugs"`
	PublishAt string   `json:"publish_at"`
}
type ListFilter struct {
	Visibility Visibility
	Docs       []DocFilter
	Search     string
	Limit      int
	Offset     int
}
type CatalogManager interface {
	CreateEntry(ctx context.Context, e Entry) (Entry, error)
	UpsertEntry(ctx context.Context, e Entry) (Entry, error)
	GetEntry(ctx context.Context, slug string) (Entry, error)
	OpenEntry(ctx context.Context, slug, key string) (Entry, error)
	UpdateEntry(ctx context.Context, e Entry) (Entry, error)
	SetVisibility(ctx context.Context, slug string, v Visibility) (Entry, error)
	SchedulePublish(ctx context.Context, slug, publishAt string) (Entry, error)
	SchedulePublishBatch(ctx context.Context, s PublishSchedule) ([]Entry, error)
	DeleteEntry(ctx context.Context, slug, deletedBy string) error
	DisposeExpired(ctx context.Context, now time.Time) ([]string, error)
	SeedEntries(ctx context.Context) ([]string, error)
	ListEntries(ctx context.Context, f ListFilter) ([]Entry, error)
	ListPublicEntries(ctx context.Context, f ListFilter) ([]Entry, error)
	PutPrivate(ctx context.Context, p EntryPrivate) (EntryPrivate, error)
	GetPrivate(ctx context.Context, slug string) (EntryPrivate, error)
	IssueShareLink(ctx context.Context, slug, label, by string) (ShareLink, string, error)
	RetireShareLink(ctx context.Context, id string) (ShareLink, error)
	ListShareLinks(ctx context.Context, slug string) ([]ShareLink, error)
	FileSuggestion(ctx context.Context, s Suggestion) (Suggestion, error)
	GetSuggestion(ctx context.Context, id string) (Suggestion, error)
	ListSuggestions(ctx context.Context, statuses ...SuggestionStatus) ([]Suggestion, error)
	DecideSuggestion(ctx context.Context, id string, status SuggestionStatus, priority *int) (Suggestion, error)
	RecordSuggestionAttempt(ctx context.Context, id, runRef, comment string) (Suggestion, error)
	CommentOnSuggestion(ctx context.Context, id string, author CommentAuthor, body string) (SuggestionComment, error)
	ListSuggestionComments(ctx context.Context, id string) ([]SuggestionComment, error)
	FulfilSuggestion(ctx context.Context, id, slug, runRef, note string) (Suggestion, error)
	AttachSuggestion(ctx context.Context, id, slug, note string) (SuggestionMatch, error)
	ListSuggestionMatches(ctx context.Context, id string) ([]SuggestionMatch, error)
	ListEntryMatches(ctx context.Context, slug string) ([]SuggestionMatch, error)
	RecordRun(ctx context.Context, r MiningRun) (MiningRun, error)
	ListRuns(ctx context.Context, limit int) ([]MiningRun, error)
	PutSetting(ctx context.Context, key, value string) (Setting, error)
	GetSetting(ctx context.Context, key string) (Setting, error)
	RecordPageView(ctx context.Context, page PageKind, slug string) error
	ListPageViews(ctx context.Context) ([]PageView, error)
	PutRecord(ctx context.Context, r Record) (Record, error)
	ListRecords(ctx context.Context, subjectID, scopeID string) ([]Record, error)
	CountRecords(ctx context.Context, subjectID, scopeID string) ([]FieldTally, error)
	Endorse(ctx context.Context, subjectID, holderID string) (int, error)
	CountEndorsements(ctx context.Context, subjectID string) (int, error)
	OfferSlot(ctx context.Context, s Slot) (Slot, error)
	ListSlots(ctx context.Context, subjectID string) ([]Slot, error)
	Book(ctx context.Context, slotID, holderID string) (Booking, error)
	CancelBooking(ctx context.Context, bookingID, by, reason string) (Booking, error)
	CancelBookingAsStaff(ctx context.Context, bookingID, by, reason string) (Booking, error)
	ListHolderBookings(ctx context.Context, holderID string) ([]Booking, error)
	ListDayBookings(ctx context.Context, subjectID, day string) ([]Booking, error)
	RaiseItem(ctx context.Context, r RaisedItem) (RaisedItem, error)
	FileRaisedItem(ctx context.Context, r RaisedItem) (RaisedItem, error)
	ListRaisedItems(ctx context.Context, raisedBy string) ([]MyRaisedItem, error)
	ReassignRaisedItem(ctx context.Context, raisedItemID, department string) error
	ListRaisedItemsOfSubject(ctx context.Context, subjectID string) ([]RaisedItem, error)
	UseAreaResolver(resolve AreaResolver)
	UseAreaDirectory(directory AreaDirectory)
	UseStreetResolver(resolve StreetResolver)
	UseLinkedEntries(linked LinkedEntries)
	NearbyRaisedItems(ctx context.Context, topic string, at geo.Position, withinMetres float64) ([]RaisedItem, error)
	PutOnBoard(ctx context.Context, b BoardItem) (BoardItem, error)
	GroupUnderBoardItem(ctx context.Context, raisedItemID, boardItemID string) error
	ListBoardItems(ctx context.Context) ([]BoardItemTally, error)
	ListPublicBoardItems(ctx context.Context, limit, offset int) ([]BoardItemTally, error)
	TallyBoardItemsByArea(ctx context.Context, boardItemID string) ([]BoardItemAreaTally, error)
	SetBoardItemOutcome(ctx context.Context, boardItemID, outcome string) error
	SetBoardItemHidden(ctx context.Context, boardItemID string, hidden bool) error
	AssignBoardItem(ctx context.Context, boardItemID, assigneeKind, assigneeID string) error
	UnassignBoardItem(ctx context.Context, boardItemID string) error
	CloseBoardItem(ctx context.Context, boardItemID, closedBy string) error
	UngroupFromBoardItem(ctx context.Context, raisedItemID string) error
	OpenSharedBoardItem(ctx context.Context, key string) (SharedBoardItem, error)
	ReplyToRaisedItem(ctx context.Context, r Reply) (Reply, error)
	ListReplies(ctx context.Context, raisedItemID string) ([]Reply, error)
	ListMyReplies(ctx context.Context, raisedItemID, caller string) ([]Reply, error)
	IssueFollowKey(ctx context.Context, raisedItemID string) (FollowKey, error)
	OpenFollowedRaisedItem(ctx context.Context, key string) (FollowedRaisedItem, error)
	ShowFollowedRaisedItem(ctx context.Context, key string) (FollowedRaisedItemView, error)
	RecordScopeFigure(ctx context.Context, f ScopeFigure) (ScopeFigure, error)
	CurrentScopeFigure(ctx context.Context, scopeID string) (ScopeFigure, error)
	ListScopeFigures(ctx context.Context, scopeID string) ([]ScopeFigure, error)
	AttachFiles(ctx context.Context, raisedItemID, raiser string, fileIDs []string) ([]RaisedItemFile, error)
	ApproveFileForPublishing(ctx context.Context, raisedItemID, fileID, approvedBy string) (RaisedItemFile, error)
	WithdrawFileApproval(ctx context.Context, raisedItemID, fileID string) (RaisedItemFile, error)
	DetachFile(ctx context.Context, raisedItemID, raiser, fileID string) error
	ListRaisedItemFiles(ctx context.Context, raisedItemID string) ([]RaisedItemFile, error)
	RemoveRaisedItem(ctx context.Context, r Removal) (Removal, error)
	ListRemovalsOfMine(ctx context.Context, raisedBy string) ([]Removal, error)
	ListOverdueRaisedItems(ctx context.Context, at string) ([]RaisedItem, error)
}
type catalogManager struct {
	db            *gorm.DB
	st            *store
	spec          *spec.Spec
	now           func() string
	resolveArea   AreaResolver
	directory     AreaDirectory
	resolveStreet StreetResolver
	linkedEntries LinkedEntries
}

func NewCatalogManager(db *gorm.DB, s *spec.Spec) (CatalogManager, error) {
	carriers := map[string]any{roleEntry: models.Entry{}, roleEntryPrivate: models.EntryPrivate{}, roleShareLink: models.ShareLink{}}
	for role, carrier := range map[string]any{roleSuggestion: models.Suggestion{}, roleSuggestionComment: models.SuggestionComment{}, roleSuggestionMatch: models.SuggestionMatch{}, roleRun: models.MiningRun{}, roleSetting: models.Setting{}, rolePageView: models.PageView{}} {
		if _, declared := s.ByRole(role); declared {
			carriers[role] = carrier
		}
	}
	if _, kept := s.ByRole(roleRecord); kept {
		carriers[roleRecord] = models.Record{}
	}
	if _, counted := s.ByRole(roleEndorsement); counted {
		carriers[roleEndorsement] = models.Endorsement{}
	}
	if _, booked := s.ByRole(roleSlot); booked {
		carriers[roleSlot] = models.Slot{}
	}
	if _, booked := s.ByRole(roleBooking); booked {
		carriers[roleBooking] = models.Booking{}
	}
	if _, taken := s.ByRole(roleRaisedItem); taken {
		carriers[roleRaisedItem] = models.RaisedItem{}
	}
	if _, boarded := s.ByRole(roleBoardItem); boarded {
		carriers[roleBoardItem] = models.BoardItem{}
	}
	if _, replied := s.ByRole(roleReply); replied {
		carriers[roleReply] = models.Reply{}
	}
	if _, attached := s.ByRole(roleRaisedItemFile); attached {
		carriers[roleRaisedItemFile] = models.RaisedItemFile{}
	}
	if _, removes := s.ByRole(roleRemoval); removes {
		carriers[roleRemoval] = models.Removal{}
	}
	if _, kept := s.ByRole(roleScopeFigure); kept {
		carriers[roleScopeFigure] = models.ScopeFigure{}
	}
	st, err := newStore(db, s, carriers)
	if err != nil {
		return nil, fmt.Errorf("NewCatalogManager: %w", err)
	}
	return &catalogManager{db: db, st: st, spec: s, now: nowRFC3339}, nil
}
func (m *catalogManager) q(ctx context.Context, role string) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.st.Table(role))
}
func (m *catalogManager) take(q *gorm.DB, role string, out any, notFound error) error {
	var rows []map[string]any
	if err := q.Limit(1).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return notFound
	}
	return decode(m.st.Object(role), rows[0], out)
}
func listOf[T any](m *catalogManager, q *gorm.DB, role string) ([]T, error) {
	var rows []map[string]any
	if err := capList(q, role).Find(&rows).Error; err != nil {
		return nil, err
	}
	o := m.st.Object(role)
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
func (m *catalogManager) insert(ctx context.Context, role string, v any) error {
	row, err := encode(m.st.Object(role), v)
	if err != nil {
		return err
	}
	return m.q(ctx, role).Create(row).Error
}
func TallyRecords(records []Record) ([]FieldTally, error) {
	return models.TallyRecords(records)
}
