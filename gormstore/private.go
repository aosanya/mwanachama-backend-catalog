package gormstore

import (
	"github.com/aosanya/mwanachama-backend-catalog/models"
)

// EntryPrivateRow is the GORM row for a [models.EntryPrivate]. Keyed by
// Slug rather than an id of its own: there is exactly one per Entry.
type EntryPrivateRow struct {
	Slug          string `gorm:"primaryKey"`
	Readme        string
	Sales         string
	ModuleConfigs JSONDoc
	Appeal        string
	Rank          int `gorm:"index"`
	UpdatedAt     string
}

// EntryPrivateToRow converts a domain EntryPrivate to its row shape.
func EntryPrivateToRow(p models.EntryPrivate) EntryPrivateRow {
	return EntryPrivateRow{
		Slug:          p.Slug,
		Readme:        p.Readme,
		Sales:         p.Sales,
		ModuleConfigs: JSONDoc(p.ModuleConfigs),
		Appeal:        p.Appeal,
		Rank:          p.Rank,
		UpdatedAt:     p.UpdatedAt,
	}
}

// RowToEntryPrivate converts a row back to its domain shape.
func RowToEntryPrivate(r EntryPrivateRow) models.EntryPrivate {
	return models.EntryPrivate{
		Slug:          r.Slug,
		Readme:        r.Readme,
		Sales:         r.Sales,
		ModuleConfigs: string(r.ModuleConfigs),
		Appeal:        r.Appeal,
		Rank:          r.Rank,
		UpdatedAt:     r.UpdatedAt,
	}
}
