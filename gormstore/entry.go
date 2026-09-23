package gormstore

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/models"
)

// EntryRow is the GORM row for a [models.Entry].
type EntryRow struct {
	ID          string `gorm:"primaryKey"`
	Slug        string `gorm:"uniqueIndex"`
	Name        string
	Summary     string
	Doc         JSONDoc
	Visibility  string `gorm:"index"`
	SubmittedBy string
	ApprovedAt  string
	CreatedAt   string
	UpdatedAt   string
	Deleted     bool
}

func (r *EntryRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// EntryToRow converts a domain Entry to its row shape.
func EntryToRow(e models.Entry) EntryRow {
	return EntryRow{
		ID:          e.ID,
		Slug:        e.Slug,
		Name:        e.Name,
		Summary:     e.Summary,
		Doc:         JSONDoc(e.Doc),
		Visibility:  string(e.Visibility),
		SubmittedBy: e.SubmittedBy,
		ApprovedAt:  e.ApprovedAt,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		Deleted:     e.Deleted,
	}
}

// RowToEntry converts a row back to its domain shape.
func RowToEntry(r EntryRow) models.Entry {
	return models.Entry{
		ID:          r.ID,
		Slug:        r.Slug,
		Name:        r.Name,
		Summary:     r.Summary,
		Doc:         string(r.Doc),
		Visibility:  models.Visibility(r.Visibility),
		SubmittedBy: r.SubmittedBy,
		ApprovedAt:  r.ApprovedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Deleted:     r.Deleted,
	}
}
