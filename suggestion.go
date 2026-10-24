package mwanachamacatalog

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/models"
)

func (m *catalogManager) suggestions(ctx context.Context) *gorm.DB {
	return m.q(ctx, roleSuggestion)
}

func (m *catalogManager) comments(ctx context.Context) *gorm.DB {
	return m.q(ctx, roleSuggestionComment)
}

func (m *catalogManager) matches(ctx context.Context) *gorm.DB {
	return m.q(ctx, roleSuggestionMatch)
}

func (m *catalogManager) findSuggestion(ctx context.Context, id string) (Suggestion, error) {
	var s Suggestion
	err := m.take(m.suggestions(ctx).Where("id = ?", id), roleSuggestion, &s, ErrSuggestionNotFound)
	return s, err
}

func (m *catalogManager) FileSuggestion(ctx context.Context, s Suggestion) (Suggestion, error) {
	if err := m.needs(roleSuggestion); err != nil {
		return Suggestion{}, err
	}
	s.Status = models.SuggestionStatusProposed
	s.Attempts = 0
	s.DecidedAt = ""
	if s.Priority == 0 {
		s.Priority = models.DefaultSuggestionPriority
	}
	if s.Source == "" {
		s.Source = models.SuggestionSourceRun
	}
	if err := m.checks(roleSuggestion, s); err != nil || s.Attempts < 0 {
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

	s.ID = newID()
	s.CreatedAt = m.now()
	if err := m.insert(ctx, roleSuggestion, s); err != nil {
		if isUniqueViolation(err) {
			return Suggestion{}, ErrSuggestionDuplicate
		}
		return Suggestion{}, err
	}
	return s, nil
}

func (m *catalogManager) GetSuggestion(ctx context.Context, id string) (Suggestion, error) {
	if err := m.needs(roleSuggestion); err != nil {
		return Suggestion{}, err
	}
	return m.findSuggestion(ctx, id)
}

func (m *catalogManager) ListSuggestions(ctx context.Context, statuses ...SuggestionStatus) ([]Suggestion, error) {
	if err := m.needs(roleSuggestion); err != nil {
		return nil, err
	}
	q := m.suggestions(ctx)
	if len(statuses) > 0 {
		in := make([]string, 0, len(statuses))
		for _, s := range statuses {
			in = append(in, string(s))
		}
		q = q.Where("status IN ?", in)
	}
	return listOf[Suggestion](m, q.Order("priority, created_at"), roleSuggestion)
}

func (m *catalogManager) DecideSuggestion(ctx context.Context, id string, status SuggestionStatus, priority int) (Suggestion, error) {
	if err := m.needs(roleSuggestion); err != nil {
		return Suggestion{}, err
	}
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
	if err := m.needs(roleSuggestion); err != nil {
		return Suggestion{}, err
	}
	// An attempt without a reason tells the next run nothing, so the count
	// would rise toward the stall with no record of why.
	if strings.TrimSpace(comment) == "" {
		return Suggestion{}, ErrAttemptNeedsComment
	}
	existing, err := m.findSuggestion(ctx, id)
	if err != nil {
		return Suggestion{}, err
	}
	if existing.Status != models.SuggestionStatusApproved {
		return Suggestion{}, ErrSuggestionStatus
	}

	attempts := existing.Attempts + 1
	status := existing.Status
	if attempts >= models.MaxSuggestionAttempts {
		status = models.SuggestionStatusStalled
	}
	if err := m.suggestions(ctx).Where("id = ?", existing.ID).Updates(map[string]any{
		"attempts": attempts, "status": string(status),
	}).Error; err != nil {
		return Suggestion{}, err
	}

	note := models.SuggestionComment{
		ID:           newID(),
		SuggestionID: existing.ID,
		Author:       models.CommentAuthorRun,
		RunID:        runID,
		Attempt:      attempts,
		Body:         strings.TrimSpace(comment),
		CreatedAt:    m.now(),
	}
	if !m.declares(roleSuggestionComment) {
		return m.GetSuggestion(ctx, id)
	}
	if err := m.insert(ctx, roleSuggestionComment, note); err != nil {
		return Suggestion{}, err
	}
	return m.GetSuggestion(ctx, id)
}

func (m *catalogManager) CommentOnSuggestion(ctx context.Context, id string, author CommentAuthor, body string) (SuggestionComment, error) {
	if err := m.needs(roleSuggestion, roleSuggestionComment); err != nil {
		return SuggestionComment{}, err
	}
	// The suggestion is read before the comment is checked, where the order
	// used to be the other way round: suggestion_id is one of the declared
	// required fields, so the comment cannot be checked as a whole until it
	// has one. A comment on a suggestion that is not there is now
	// ErrSuggestionNotFound rather than ErrInvalidComment, which is the more
	// accurate of the two answers anyway.
	existing, err := m.findSuggestion(ctx, id)
	if err != nil {
		return SuggestionComment{}, err
	}
	c := models.SuggestionComment{
		ID:           newID(),
		SuggestionID: existing.ID,
		Author:       author,
		Body:         strings.TrimSpace(body),
		CreatedAt:    m.now(),
	}
	if err := m.checks(roleSuggestionComment, c); err != nil {
		return SuggestionComment{}, ErrInvalidComment
	}
	if err := m.insert(ctx, roleSuggestionComment, c); err != nil {
		return SuggestionComment{}, err
	}
	return c, nil
}

func (m *catalogManager) ListSuggestionComments(ctx context.Context, id string) ([]SuggestionComment, error) {
	if err := m.needs(roleSuggestion, roleSuggestionComment); err != nil {
		return nil, err
	}
	if _, err := m.findSuggestion(ctx, id); err != nil {
		return nil, err
	}
	return listOf[SuggestionComment](m,
		m.comments(ctx).Where("suggestion_id = ?", id).Order("created_at, id"),
		roleSuggestionComment)
}

func (m *catalogManager) upsertMatch(ctx context.Context, suggestionID, slug, runID, note string) (SuggestionMatch, error) {
	if _, err := m.findEntry(ctx, slug); err != nil {
		return SuggestionMatch{}, err
	}
	match := models.SuggestionMatch{
		SuggestionID: suggestionID, Slug: slug, RunID: runID, Note: note, MetAt: m.now(),
	}
	var n int64
	if err := m.matches(ctx).Where("suggestion_id = ? AND slug = ?", suggestionID, slug).
		Count(&n).Error; err != nil {
		return SuggestionMatch{}, err
	}
	if n > 0 {
		if err := m.matches(ctx).Where("suggestion_id = ? AND slug = ?", suggestionID, slug).
			Updates(map[string]any{"run_id": runID, "note": note, "met_at": match.MetAt}).Error; err != nil {
			return SuggestionMatch{}, err
		}
		return match, nil
	}
	if err := m.insert(ctx, roleSuggestionMatch, match); err != nil {
		return SuggestionMatch{}, err
	}
	return match, nil
}

func (m *catalogManager) AttachSuggestion(ctx context.Context, id, slug, note string) (SuggestionMatch, error) {
	if err := m.needs(roleSuggestion, roleSuggestionMatch); err != nil {
		return SuggestionMatch{}, err
	}
	existing, err := m.findSuggestion(ctx, id)
	if err != nil {
		return SuggestionMatch{}, err
	}
	return m.upsertMatch(ctx, existing.ID, slug, existing.RunID, note)
}

func (m *catalogManager) FulfilSuggestion(ctx context.Context, id, slug, runID, note string) (Suggestion, error) {
	if err := m.needs(roleSuggestion, roleSuggestionMatch); err != nil {
		return Suggestion{}, err
	}
	existing, err := m.findSuggestion(ctx, id)
	if err != nil {
		return Suggestion{}, err
	}
	switch existing.Status {
	case models.SuggestionStatusApproved, models.SuggestionStatusBuilt:
	default:
		return Suggestion{}, ErrSuggestionStatus
	}
	if _, err := m.upsertMatch(ctx, existing.ID, slug, runID, note); err != nil {
		return Suggestion{}, err
	}
	if err := m.suggestions(ctx).Where("id = ?", existing.ID).
		Update("status", string(models.SuggestionStatusBuilt)).Error; err != nil {
		return Suggestion{}, err
	}
	return m.GetSuggestion(ctx, id)
}

func (m *catalogManager) ListSuggestionMatches(ctx context.Context, id string) ([]SuggestionMatch, error) {
	if err := m.needs(roleSuggestion, roleSuggestionMatch); err != nil {
		return nil, err
	}
	if _, err := m.findSuggestion(ctx, id); err != nil {
		return nil, err
	}
	return listOf[SuggestionMatch](m,
		m.matches(ctx).Where("suggestion_id = ?", id).Order("slug"),
		roleSuggestionMatch)
}

func (m *catalogManager) ListEntryMatches(ctx context.Context, slug string) ([]SuggestionMatch, error) {
	if err := m.needs(roleSuggestionMatch); err != nil {
		return nil, err
	}
	if _, err := m.findEntry(ctx, slug); err != nil {
		return nil, err
	}
	return listOf[SuggestionMatch](m,
		m.matches(ctx).Where("slug = ?", slug).Order("met_at, suggestion_id"),
		roleSuggestionMatch)
}
