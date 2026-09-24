package mwanachamacatalog

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

func (m *catalogManager) privates(ctx context.Context) *gorm.DB {
	return m.q(ctx, roleEntryPrivate)
}

func (m *catalogManager) PutPrivate(ctx context.Context, p EntryPrivate) (EntryPrivate, error) {
	if err := m.checks(roleEntryPrivate, p); err != nil || p.Rank < 0 {
		return EntryPrivate{}, ErrInvalidPrivate
	}
	if _, err := m.findEntry(ctx, p.Slug); err != nil {
		return EntryPrivate{}, err
	}

	var existing EntryPrivate
	err := m.take(m.privates(ctx).Where("slug = ?", p.Slug), roleEntryPrivate, &existing, errNoPrivate)
	switch {
	case errors.Is(err, errNoPrivate):
		p.UpdatedAt = m.now()
		if err := m.insert(ctx, roleEntryPrivate, p); err != nil {
			return EntryPrivate{}, err
		}
		return p, nil
	case err != nil:
		return EntryPrivate{}, err
	}

	// A partial write: an empty field means "leave it alone", so a caller
	// re-ranking the catalog does not have to resend every readme to avoid
	// erasing them.
	merged := existing
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

	row, err := encode(m.st.Object(roleEntryPrivate), merged)
	if err != nil {
		return EntryPrivate{}, err
	}
	if err := m.privates(ctx).Where("slug = ?", p.Slug).Updates(row).Error; err != nil {
		return EntryPrivate{}, err
	}
	return merged, nil
}

func (m *catalogManager) GetPrivate(ctx context.Context, slug string) (EntryPrivate, error) {
	if _, err := m.findEntry(ctx, slug); err != nil {
		return EntryPrivate{}, err
	}

	var p EntryPrivate
	err := m.take(m.privates(ctx).Where("slug = ?", slug), roleEntryPrivate, &p, errNoPrivate)
	if errors.Is(err, errNoPrivate) {
		// An Entry with nothing written about it yet is not an error; it
		// has empty private parts.
		return EntryPrivate{Slug: slug}, nil
	}
	if err != nil {
		return EntryPrivate{}, err
	}
	return p, nil
}

// errNoPrivate is internal: both callers turn it into something else, and
// neither ever lets it out.
var errNoPrivate = errors.New("catalog: no private parts stored")
