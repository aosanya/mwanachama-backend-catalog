package gormstore

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/models"
)

// SuggestionRow is the GORM row for a [models.Suggestion].
type SuggestionRow struct {
	ID        string `gorm:"primaryKey"`
	Slug      string `gorm:"uniqueIndex"`
	Title     string
	Why       string
	Source    string
	Status    string `gorm:"index:idx_status_priority,priority:1"`
	Priority  int    `gorm:"index:idx_status_priority,priority:2"`
	Attempts  int
	Proposal  JSONDoc
	RunID     string
	CreatedAt string
	DecidedAt string
}

func (r *SuggestionRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// SuggestionToRow converts a domain Suggestion to its row shape.
func SuggestionToRow(s models.Suggestion) SuggestionRow {
	return SuggestionRow{
		ID:        s.ID,
		Slug:      s.Slug,
		Title:     s.Title,
		Why:       s.Why,
		Source:    string(s.Source),
		Status:    string(s.Status),
		Priority:  s.Priority,
		Attempts:  s.Attempts,
		Proposal:  JSONDoc(s.Proposal),
		RunID:     s.RunID,
		CreatedAt: s.CreatedAt,
		DecidedAt: s.DecidedAt,
	}
}

// RowToSuggestion converts a row back to its domain shape.
func RowToSuggestion(r SuggestionRow) models.Suggestion {
	return models.Suggestion{
		ID:        r.ID,
		Slug:      r.Slug,
		Title:     r.Title,
		Why:       r.Why,
		Source:    models.SuggestionSource(r.Source),
		Status:    models.SuggestionStatus(r.Status),
		Priority:  r.Priority,
		Attempts:  r.Attempts,
		Proposal:  string(r.Proposal),
		RunID:     r.RunID,
		CreatedAt: r.CreatedAt,
		DecidedAt: r.DecidedAt,
	}
}

// SuggestionCommentRow is the GORM row for a [models.SuggestionComment].
type SuggestionCommentRow struct {
	ID           string `gorm:"primaryKey"`
	SuggestionID string `gorm:"index:idx_suggestion_created,priority:1"`
	Author       string
	RunID        string
	Attempt      int
	Body         string
	CreatedAt    string `gorm:"index:idx_suggestion_created,priority:2"`
}

func (r *SuggestionCommentRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// SuggestionCommentToRow converts a domain SuggestionComment to its row shape.
func SuggestionCommentToRow(c models.SuggestionComment) SuggestionCommentRow {
	return SuggestionCommentRow{
		ID:           c.ID,
		SuggestionID: c.SuggestionID,
		Author:       string(c.Author),
		RunID:        c.RunID,
		Attempt:      c.Attempt,
		Body:         c.Body,
		CreatedAt:    c.CreatedAt,
	}
}

// RowToSuggestionComment converts a row back to its domain shape.
func RowToSuggestionComment(r SuggestionCommentRow) models.SuggestionComment {
	return models.SuggestionComment{
		ID:           r.ID,
		SuggestionID: r.SuggestionID,
		Author:       models.CommentAuthor(r.Author),
		RunID:        r.RunID,
		Attempt:      r.Attempt,
		Body:         r.Body,
		CreatedAt:    r.CreatedAt,
	}
}

// SuggestionMatchRow is the GORM row for a [models.SuggestionMatch], keyed
// by the pair so one Entry can only answer a suggestion once.
type SuggestionMatchRow struct {
	SuggestionID string `gorm:"primaryKey"`
	Slug         string `gorm:"primaryKey;index"`
	RunID        string
	Note         string
	MetAt        string
}

// SuggestionMatchToRow converts a domain SuggestionMatch to its row shape.
func SuggestionMatchToRow(m models.SuggestionMatch) SuggestionMatchRow {
	return SuggestionMatchRow{
		SuggestionID: m.SuggestionID,
		Slug:         m.Slug,
		RunID:        m.RunID,
		Note:         m.Note,
		MetAt:        m.MetAt,
	}
}

// RowToSuggestionMatch converts a row back to its domain shape.
func RowToSuggestionMatch(r SuggestionMatchRow) models.SuggestionMatch {
	return models.SuggestionMatch{
		SuggestionID: r.SuggestionID,
		Slug:         r.Slug,
		RunID:        r.RunID,
		Note:         r.Note,
		MetAt:        r.MetAt,
	}
}
