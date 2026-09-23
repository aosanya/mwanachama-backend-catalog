package mwanachamacatalog

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/gormstore"
)

func (m *catalogManager) privates(ctx context.Context) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.tables.EntryPrivates)
}

func (m *catalogManager) PutPrivate(ctx context.Context, p EntryPrivate) (EntryPrivate, error) {
	if !p.Validate() {
		return EntryPrivate{}, ErrInvalidPrivate
	}
	if _, err := m.findEntry(ctx, p.Slug); err != nil {
		return EntryPrivate{}, err
	}

	var existing gormstore.EntryPrivateRow
	err := m.privates(ctx).Where("slug = ?", p.Slug).Take(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		p.UpdatedAt = m.now()
		row := gormstore.EntryPrivateToRow(p)
		if err := m.privates(ctx).Create(&row).Error; err != nil {
			return EntryPrivate{}, err
		}
		return gormstore.RowToEntryPrivate(row), nil
	case err != nil:
		return EntryPrivate{}, err
	}

	// A partial write: an empty field means "leave it alone", so a caller
	// re-ranking the catalog does not have to resend every readme to avoid
	// erasing them.
	merged := gormstore.RowToEntryPrivate(existing)
	if p.Readme != "" {
		merged.Readme = p.Readme
	}
	if p.Sales != "" {
		merged.Sales = p.Sales
	}
	if p.ModuleConfigs != "" {
		merged.ModuleConfigs = p.ModuleConfigs
	}
	if p.Appeal != "" {
		merged.Appeal = p.Appeal
	}
	if p.Rank != 0 {
		merged.Rank = p.Rank
	}
	merged.UpdatedAt = m.now()

	row := gormstore.EntryPrivateToRow(merged)
	if err := m.privates(ctx).Where("slug = ?", p.Slug).Save(&row).Error; err != nil {
		return EntryPrivate{}, err
	}
	return gormstore.RowToEntryPrivate(row), nil
}

func (m *catalogManager) GetPrivate(ctx context.Context, slug string) (EntryPrivate, error) {
	if _, err := m.findEntry(ctx, slug); err != nil {
		return EntryPrivate{}, err
	}

	var row gormstore.EntryPrivateRow
	err := m.privates(ctx).Where("slug = ?", slug).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// An Entry with nothing written about it yet is not an error; it
		// has empty private parts.
		return EntryPrivate{Slug: slug}, nil
	}
	if err != nil {
		return EntryPrivate{}, err
	}
	return gormstore.RowToEntryPrivate(row), nil
}
