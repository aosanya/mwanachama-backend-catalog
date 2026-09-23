package gormstore

import (
	"gorm.io/gorm"
)

// TableNames configures which physical tables a CatalogManager reads and
// writes.
type TableNames struct {
	Entries            string
	EntryPrivates      string
	ShareLinks         string
	Suggestions        string
	SuggestionComments string
	SuggestionMatches  string
	MiningRuns         string
	Settings           string
	PageViews          string
}

// DefaultTableNames builds the conventional table set for one mounted
// instance of this package, e.g. DefaultTableNames("wakala") yields
// wakala_entries, wakala_entry_privates, wakala_share_links and the rest.
func DefaultTableNames(instance string) TableNames {
	return TableNames{
		Entries:            instance + "_entries",
		EntryPrivates:      instance + "_entry_privates",
		ShareLinks:         instance + "_share_links",
		Suggestions:        instance + "_suggestions",
		SuggestionComments: instance + "_suggestion_comments",
		SuggestionMatches:  instance + "_suggestion_matches",
		MiningRuns:         instance + "_mining_runs",
		Settings:           instance + "_settings",
		PageViews:          instance + "_page_views",
	}
}

// Migrate creates or updates the tables t names. Callers run this once at
// startup, or in test setup, before constructing a CatalogManager with the
// same db and t.
func Migrate(db *gorm.DB, t TableNames) error {
	for _, m := range []struct {
		table string
		model any
	}{
		{t.Entries, &EntryRow{}},
		{t.EntryPrivates, &EntryPrivateRow{}},
		{t.ShareLinks, &ShareLinkRow{}},
		{t.Suggestions, &SuggestionRow{}},
		{t.SuggestionComments, &SuggestionCommentRow{}},
		{t.SuggestionMatches, &SuggestionMatchRow{}},
		{t.MiningRuns, &MiningRunRow{}},
		{t.Settings, &SettingRow{}},
		{t.PageViews, &PageViewRow{}},
	} {
		if err := db.Table(m.table).AutoMigrate(m.model); err != nil {
			return err
		}
	}
	return nil
}
