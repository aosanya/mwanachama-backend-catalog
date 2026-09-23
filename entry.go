package mwanachamacatalog

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/gormstore"
	"github.com/aosanya/mwanachama-backend-catalog/models"
)

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

func (m *catalogManager) entries(ctx context.Context) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.tables.Entries)
}

// findEntry reads one non-deleted row by slug.
func (m *catalogManager) findEntry(ctx context.Context, slug string) (gormstore.EntryRow, error) {
	var row gormstore.EntryRow
	err := m.entries(ctx).Where("slug = ? AND deleted = ?", slug, false).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.EntryRow{}, ErrEntryNotFound
	}
	return row, err
}

func (m *catalogManager) CreateEntry(ctx context.Context, e Entry) (Entry, error) {
	e.Visibility = models.VisibilityPending
	e.ApprovedAt = ""
	if !e.Validate() {
		return Entry{}, ErrInvalidEntry
	}

	now := m.now()
	e.ID = ""
	e.CreatedAt, e.UpdatedAt = now, now
	e.Deleted = false

	row := gormstore.EntryToRow(e)
	if err := m.entries(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return Entry{}, ErrEntryExists
		}
		return Entry{}, err
	}
	return gormstore.RowToEntry(row), nil
}

func (m *catalogManager) UpsertEntry(ctx context.Context, e Entry) (Entry, error) {
	existing, err := m.findEntry(ctx, e.Slug)
	if errors.Is(err, ErrEntryNotFound) {
		return m.CreateEntry(ctx, e)
	}
	if err != nil {
		return Entry{}, err
	}

	// A changed document has not been reviewed, so it goes back to pending
	// however the stored row was approved. An unchanged one leaves the
	// review alone, which is what lets a mining run re-send an agency it
	// already published without un-publishing it.
	e.Visibility = models.Visibility(existing.Visibility)
	e.ApprovedAt = existing.ApprovedAt
	if string(existing.Doc) != e.Doc {
		e.Visibility = models.VisibilityPending
		e.ApprovedAt = ""
	}
	if !e.Validate() {
		return Entry{}, ErrInvalidEntry
	}

	e.ID = existing.ID
	e.CreatedAt = existing.CreatedAt
	e.UpdatedAt = m.now()
	e.Deleted = false

	row := gormstore.EntryToRow(e)
	if err := m.entries(ctx).Where("id = ?", existing.ID).Save(&row).Error; err != nil {
		return Entry{}, err
	}
	return gormstore.RowToEntry(row), nil
}

func (m *catalogManager) GetEntry(ctx context.Context, slug string) (Entry, error) {
	row, err := m.findEntry(ctx, slug)
	if err != nil {
		return Entry{}, err
	}
	return gormstore.RowToEntry(row), nil
}

func (m *catalogManager) OpenEntry(ctx context.Context, slug, key string) (Entry, error) {
	row, err := m.findEntry(ctx, slug)
	if err != nil {
		return Entry{}, err
	}

	switch models.Visibility(row.Visibility) {
	case models.VisibilityPublic:
		return gormstore.RowToEntry(row), nil
	case models.VisibilityPrivate:
		if key == "" {
			return Entry{}, ErrEntryNotFound
		}
		ok, err := m.keyOpens(ctx, slug, key)
		if err != nil {
			return Entry{}, err
		}
		if !ok {
			return Entry{}, ErrEntryNotFound
		}
		return gormstore.RowToEntry(row), nil
	default:
		return Entry{}, ErrEntryNotFound
	}
}

func (m *catalogManager) UpdateEntry(ctx context.Context, e Entry) (Entry, error) {
	existing, err := m.findEntry(ctx, e.Slug)
	if err != nil {
		return Entry{}, err
	}
	if e.ID != "" && e.ID != existing.ID {
		return Entry{}, ErrSlugImmutable
	}
	if !e.Validate() {
		return Entry{}, ErrInvalidEntry
	}

	e.ID = existing.ID
	e.CreatedAt = existing.CreatedAt
	e.UpdatedAt = m.now()
	e.Deleted = false

	row := gormstore.EntryToRow(e)
	if err := m.entries(ctx).Where("id = ?", existing.ID).Save(&row).Error; err != nil {
		return Entry{}, err
	}
	return gormstore.RowToEntry(row), nil
}

func (m *catalogManager) SetVisibility(ctx context.Context, slug string, v Visibility) (Entry, error) {
	if !models.IsVisibility(v) {
		return Entry{}, ErrInvalidEntry
	}
	existing, err := m.findEntry(ctx, slug)
	if err != nil {
		return Entry{}, err
	}

	now := m.now()
	approvedAt := now
	if v == models.VisibilityPending {
		approvedAt = ""
	}

	updates := map[string]any{
		"visibility":  string(v),
		"approved_at": approvedAt,
		"updated_at":  now,
	}
	if err := m.entries(ctx).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
		return Entry{}, err
	}

	// Retiring the keys is what makes "no longer private" mean something.
	// A link left active would keep opening the Entry if it were ever made
	// private again, by someone who was handed it under the old review.
	if v != models.VisibilityPrivate {
		if err := m.retireLinksFor(ctx, slug, now); err != nil {
			return Entry{}, err
		}
	}
	return m.GetEntry(ctx, slug)
}

func (m *catalogManager) DeleteEntry(ctx context.Context, slug string) error {
	existing, err := m.findEntry(ctx, slug)
	if err != nil {
		return err
	}
	now := m.now()
	if err := m.entries(ctx).Where("id = ?", existing.ID).
		Updates(map[string]any{"deleted": true, "updated_at": now}).Error; err != nil {
		return err
	}
	return m.retireLinksFor(ctx, slug, now)
}

func (m *catalogManager) ListEntries(ctx context.Context, f ListFilter) ([]Entry, error) {
	q := m.entries(ctx).Where("deleted = ?", false)
	if f.Visibility != "" {
		q = q.Where("visibility = ?", string(f.Visibility))
	}
	for _, d := range f.Docs {
		if !d.Valid() {
			return nil, ErrInvalidDocPath
		}
		q = q.Where(docPathExpr(m.db, d)+" = ?", d.Value)
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		q = q.Where("LOWER(name) LIKE ? OR LOWER(summary) LIKE ?", like, like)
	}
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}

	var rows []gormstore.EntryRow
	if err := q.Order("name").Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]Entry, 0, len(rows))
	for _, r := range rows {
		e := gormstore.RowToEntry(r)
		// A listing is the card. Carrying Doc would put 34 KB per row into
		// every storefront index page.
		e.Doc = ""
		out = append(out, e)
	}
	return out, nil
}

// docPathExpr builds the SQL that reads one path out of the document, in
// the dialect in use. Neither dialect takes the path as a bound parameter,
// which is why [models.DocFilter.Valid] restricts it to an alphabet with no
// quote, brace or bracket in it — callers must check Valid before reaching
// here, and ListEntries does.
func docPathExpr(db *gorm.DB, d models.DocFilter) string {
	segs := d.Segments()
	if db.Dialector.Name() == "postgres" {
		return `doc #>> '{` + strings.Join(segs, ",") + `}'`
	}
	return `json_extract(doc, '$.` + strings.Join(segs, ".") + `')`
}

// isUniqueViolation reports whether err is a duplicate-key error. The two
// drivers word it differently and neither exposes a portable code through
// GORM.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "unique") || strings.Contains(s, "duplicate")
}
