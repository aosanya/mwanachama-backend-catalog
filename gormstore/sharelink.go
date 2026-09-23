package gormstore

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/models"
)

// ShareLinkRow is the GORM row for a [models.ShareLink]. KeyHash is unique
// across the whole table, not per Entry: a key that opened two different
// Entries would be a collision in the one namespace callers present keys
// from.
type ShareLinkRow struct {
	ID        string `gorm:"primaryKey"`
	Slug      string `gorm:"index"`
	KeyHash   string `gorm:"uniqueIndex"`
	Label     string
	Status    string `gorm:"index"`
	CreatedBy string
	CreatedAt string
	RetiredAt string
}

func (r *ShareLinkRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// ShareLinkToRow converts a domain ShareLink to its row shape.
func ShareLinkToRow(l models.ShareLink) ShareLinkRow {
	return ShareLinkRow{
		ID:        l.ID,
		Slug:      l.Slug,
		KeyHash:   l.KeyHash,
		Label:     l.Label,
		Status:    string(l.Status),
		CreatedBy: l.CreatedBy,
		CreatedAt: l.CreatedAt,
		RetiredAt: l.RetiredAt,
	}
}

// RowToShareLink converts a row back to its domain shape.
func RowToShareLink(r ShareLinkRow) models.ShareLink {
	return models.ShareLink{
		ID:        r.ID,
		Slug:      r.Slug,
		KeyHash:   r.KeyHash,
		Label:     r.Label,
		Status:    models.LinkStatus(r.Status),
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
		RetiredAt: r.RetiredAt,
	}
}
