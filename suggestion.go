package mwanachamacatalog

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/gormstore"
	"github.com/aosanya/mwanachama-backend-catalog/models"
)

func (m *catalogManager) suggestions(ctx context.Context) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.tables.Suggestions)
}

func (m *catalogManager) comments(ctx context.Context) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.tables.SuggestionComments)
}

func (m *catalogManager) matches(ctx context.Context) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.tables.SuggestionMatches)
}

func (m *catalogManager) findSuggestion(ctx context.Context, id string) (gormstore.SuggestionRow, error) {
	var row gormstore.SuggestionRow
	err := m.suggestions(ctx).Where("id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.SuggestionRow{}, ErrSuggestionNotFound
	}
	return row, err
}

func (m *catalogManager) FileSuggestion(ctx context.Context, s Suggestion) (Suggestion, error) {
	s.Status = models.SuggestionStatusProposed
	s.Attempts = 0
	s.DecidedAt = ""
	if s.Priority == 0 {
		s.Priority = models.DefaultSuggestionPriority
	}
	if s.Source == "" {
		s.Source = models.SuggestionSourceRun
	}
	if !s.Validate() {
		return Suggestion{}, ErrInvalidSuggestion
	}

	// Dedup is not the caller's to skip: a slug that is already an Entry or
	// already proposed is refused here rather than filed twice.
	if _, err := m.findEntry(ctx, s.Slug); err == nil {
		return Suggestion{}, ErrSuggestionDuplicate
	} else if !errors.Is(err, ErrEntryNotFound) {
		return Suggestion{}, err
	}
	var n int64
	if err := m.suggestions(ctx).Where("slug = ?", s.Slug).Count(&n).Error; err != nil {
		return Suggestion{}, err
	}
	if n > 0 {
		return Suggestion{}, ErrSuggestionDuplicate
	}

	s.ID = ""
	s.CreatedAt = m.now()
	row := gormstore.SuggestionToRow(s)
	if err := m.suggestions(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return Suggestion{}, ErrSuggestionDuplicate
		}
		return Suggestion{}, err
	}
	return gormstore.RowToSuggestion(row), nil
}

func (m *catalogManager) GetSuggestion(ctx context.Context, id string) (Suggestion, error) {
	row, err := m.findSuggestion(ctx, id)
	if err != nil {
		return Suggestion{}, err
	}
	return gormstore.RowToSuggestion(row), nil
}

func (m *catalogManager) ListSuggestions(ctx context.Context, statuses ...SuggestionStatus) ([]Suggestion, error) {
	q := m.suggestions(ctx)
	if len(statuses) > 0 {
		in := make([]string, 0, len(statuses))
		for _, s := range statuses {
			in = append(in, string(s))
		}
		q = q.Where("status IN ?", in)
	}
	var rows []gormstore.SuggestionRow
	if err := q.Order("priority, created_at").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Suggestion, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.RowToSuggestion(r))
	}
	return out, nil
}

func (m *catalogManager) DecideSuggestion(ctx context.Context, id string, status SuggestionStatus, priority int) (Suggestion, error) {
	// Built is reached by fulfilling a suggestion, never by deciding one —
	// otherwise the catalog would claim an archetype exists that nothing
	// built.
	switch status {
	case models.SuggestionStatusProposed, models.SuggestionStatusApproved, models.SuggestionStatusRejected:
	default:
		return Suggestion{}, ErrSuggestionStatus
	}
	existing, err := m.findSuggestion(ctx, id)
	if err != nil {
		return Suggestion{}, err
	}

	updates := map[string]any{"status": string(status), "decided_at": m.now()}
	if status == models.SuggestionStatusProposed {
		updates["decided_at"] = ""
	}
	// Approving clears the attempts, which is what lets an admin send a
	// stalled suggestion back for another ten tries.
	if status == models.SuggestionStatusApproved {
		updates["attempts"] = 0
	}
	if priority > 0 {
		updates["priority"] = priority
	}
	if err := m.suggestions(ctx).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
		return Suggestion{}, err
	}
	return m.GetSuggestion(ctx, id)
}

func (m *catalogManager) RecordSuggestionAttempt(ctx context.Context, id, runID, comment string) (Suggestion, error) {
	// An attempt without a reason tells the next run nothing, so the count
	// would rise toward the stall with no record of why.
	if strings.TrimSpace(comment) == "" {
		return Suggestion{}, ErrAttemptNeedsComment
	}
	existing, err := m.findSuggestion(ctx, id)
	if err != nil {
		return Suggestion{}, err
	}
	if models.SuggestionStatus(existing.Status) != models.SuggestionStatusApproved {
		return Suggestion{}, ErrSuggestionStatus
	}

	attempts := existing.Attempts + 1
	status := models.SuggestionStatus(existing.Status)
	if attempts >= models.MaxSuggestionAttempts {
		status = models.SuggestionStatusStalled
	}
	if err := m.suggestions(ctx).Where("id = ?", existing.ID).Updates(map[string]any{
		"attempts": attempts, "status": string(status),
	}).Error; err != nil {
		return Suggestion{}, err
	}

	note := gormstore.SuggestionCommentToRow(models.SuggestionComment{
		SuggestionID: existing.ID,
		Author:       models.CommentAuthorRun,
		RunID:        runID,
		Attempt:      attempts,
		Body:         strings.TrimSpace(comment),
		CreatedAt:    m.now(),
	})
	if err := m.comments(ctx).Create(&note).Error; err != nil {
		return Suggestion{}, err
	}
	return m.GetSuggestion(ctx, id)
}

func (m *catalogManager) CommentOnSuggestion(ctx context.Context, id string, author CommentAuthor, body string) (SuggestionComment, error) {
	if strings.TrimSpace(body) == "" || !models.IsCommentAuthor(author) {
		return SuggestionComment{}, ErrInvalidComment
	}
	existing, err := m.findSuggestion(ctx, id)
	if err != nil {
		return SuggestionComment{}, err
	}
	row := gormstore.SuggestionCommentToRow(models.SuggestionComment{
		SuggestionID: existing.ID,
		Author:       author,
		Body:         strings.TrimSpace(body),
		CreatedAt:    m.now(),
	})
	if err := m.comments(ctx).Create(&row).Error; err != nil {
		return SuggestionComment{}, err
	}
	return gormstore.RowToSuggestionComment(row), nil
}

func (m *catalogManager) ListSuggestionComments(ctx context.Context, id string) ([]SuggestionComment, error) {
	if _, err := m.findSuggestion(ctx, id); err != nil {
		return nil, err
	}
	var rows []gormstore.SuggestionCommentRow
	if err := m.comments(ctx).Where("suggestion_id = ?", id).
		Order("created_at, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]SuggestionComment, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.RowToSuggestionComment(r))
	}
	return out, nil
}

func (m *catalogManager) FulfilSuggestion(ctx context.Context, id, slug, runID, note string) (Suggestion, error) {
	existing, err := m.findSuggestion(ctx, id)
	if err != nil {
		return Suggestion{}, err
	}
	switch models.SuggestionStatus(existing.Status) {
	case models.SuggestionStatusApproved, models.SuggestionStatusBuilt:
	default:
		return Suggestion{}, ErrSuggestionStatus
	}
	// The Entry has to exist, or the suggestion would be marked built with
	// nothing answering it.
	if _, err := m.findEntry(ctx, slug); err != nil {
		return Suggestion{}, err
	}

	match := models.SuggestionMatch{
		SuggestionID: existing.ID, Slug: slug, RunID: runID, Note: note, MetAt: m.now(),
	}
	row := gormstore.SuggestionMatchToRow(match)
	var n int64
	if err := m.matches(ctx).Where("suggestion_id = ? AND slug = ?", existing.ID, slug).
		Count(&n).Error; err != nil {
		return Suggestion{}, err
	}
	if n > 0 {
		if err := m.matches(ctx).Where("suggestion_id = ? AND slug = ?", existing.ID, slug).
			Updates(map[string]any{"run_id": runID, "note": note, "met_at": match.MetAt}).Error; err != nil {
			return Suggestion{}, err
		}
	} else if err := m.matches(ctx).Create(&row).Error; err != nil {
		return Suggestion{}, err
	}

	if err := m.suggestions(ctx).Where("id = ?", existing.ID).
		Update("status", string(models.SuggestionStatusBuilt)).Error; err != nil {
		return Suggestion{}, err
	}
	return m.GetSuggestion(ctx, id)
}

func (m *catalogManager) ListSuggestionMatches(ctx context.Context, id string) ([]SuggestionMatch, error) {
	if _, err := m.findSuggestion(ctx, id); err != nil {
		return nil, err
	}
	var rows []gormstore.SuggestionMatchRow
	if err := m.matches(ctx).Where("suggestion_id = ?", id).Order("slug").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]SuggestionMatch, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.RowToSuggestionMatch(r))
	}
	return out, nil
}
