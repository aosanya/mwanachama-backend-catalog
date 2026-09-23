package gormstore

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/models"
)

// MiningRunRow is the GORM row for a [models.MiningRun].
type MiningRunRow struct {
	ID             string `gorm:"primaryKey"`
	RunID          string
	SubmittedBy    string
	TemplatesFiled int
	Summary        string
	CreatedAt      string `gorm:"index"`
}

func (r *MiningRunRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// MiningRunToRow converts a domain MiningRun to its row shape.
func MiningRunToRow(r models.MiningRun) MiningRunRow {
	return MiningRunRow{
		ID:             r.ID,
		RunID:          r.RunID,
		SubmittedBy:    r.SubmittedBy,
		TemplatesFiled: r.TemplatesFiled,
		Summary:        r.Summary,
		CreatedAt:      r.CreatedAt,
	}
}

// RowToMiningRun converts a row back to its domain shape.
func RowToMiningRun(r MiningRunRow) models.MiningRun {
	return models.MiningRun{
		ID:             r.ID,
		RunID:          r.RunID,
		SubmittedBy:    r.SubmittedBy,
		TemplatesFiled: r.TemplatesFiled,
		Summary:        r.Summary,
		CreatedAt:      r.CreatedAt,
	}
}

// SettingRow is the GORM row for a [models.Setting].
type SettingRow struct {
	Key       string `gorm:"primaryKey"`
	Value     string
	UpdatedAt string
}

// SettingToRow converts a domain Setting to its row shape.
func SettingToRow(s models.Setting) SettingRow {
	return SettingRow{Key: s.Key, Value: s.Value, UpdatedAt: s.UpdatedAt}
}

// RowToSetting converts a row back to its domain shape.
func RowToSetting(r SettingRow) models.Setting {
	return models.Setting{Key: r.Key, Value: r.Value, UpdatedAt: r.UpdatedAt}
}

// PageViewRow is the GORM row for a [models.PageView], keyed by the pair so
// each page has exactly one running total.
type PageViewRow struct {
	Page  string `gorm:"primaryKey"`
	Slug  string `gorm:"primaryKey"`
	Views int64
}

// PageViewToRow converts a domain PageView to its row shape.
func PageViewToRow(v models.PageView) PageViewRow {
	return PageViewRow{Page: string(v.Page), Slug: v.Slug, Views: v.Views}
}

// RowToPageView converts a row back to its domain shape.
func RowToPageView(r PageViewRow) models.PageView {
	return models.PageView{Page: models.PageKind(r.Page), Slug: r.Slug, Views: r.Views}
}
