package mwanachamacatalog

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

func (m *catalogManager) checkEntry(e *Entry) error {
	at, ok := models.NormalizePublishAt(e.PublishAt)
	if !ok {
		return ErrInvalidEntry
	}
	e.PublishAt = at
	disposeAt, ok := models.NormalizePublishAt(e.DisposeAt)
	if !ok {
		return ErrInvalidEntry
	}
	e.DisposeAt = disposeAt
	if err := m.checks(roleEntry, *e); err != nil {
		return ErrInvalidEntry
	}
	return nil
}

func (m *catalogManager) entries(ctx context.Context) *gorm.DB {
	return m.q(ctx, roleEntry)
}

// findEntry reads one non-deleted Entry by slug.
func (m *catalogManager) findEntry(ctx context.Context, slug string) (Entry, error) {
	var e Entry
	err := m.take(
		m.entries(ctx).Where("slug = ? AND deleted = ?", slug, false),
		roleEntry, &e, ErrEntryNotFound)
	return e, err
}

func (m *catalogManager) CreateEntry(ctx context.Context, e Entry) (Entry, error) {
	e.Visibility = models.VisibilityPending
	e.ApprovedAt = ""
	if err := m.checkEntry(&e); err != nil {
		return Entry{}, err
	}
	if err := m.checkKind(ctx, e.Doc); err != nil {
		return Entry{}, err
	}

	now := m.now()
	e.ID = newID()
	e.CreatedAt, e.UpdatedAt = now, now
	e.Deleted = false
	if err := m.insert(ctx, roleEntry, e); err != nil {
		if isUniqueViolation(err) {
			return Entry{}, ErrEntryExists
		}
		return Entry{}, err
	}
	return e, nil
}

func (m *catalogManager) UpsertEntry(ctx context.Context, e Entry) (Entry, error) {
	existing, err := m.findEntry(ctx, e.Slug)
	if errors.Is(err, ErrEntryNotFound) {
		e.PublishAt = ""
		return m.CreateEntry(ctx, e)
	}
	if err != nil {
		return Entry{}, err
	}
	e.PublishAt = existing.PublishAt
	return m.rewriteEntry(ctx, existing, e)
}

func reviewedContentChanged(existing, sent Entry) bool {
	return !sameDocument(existing.Doc, sent.Doc) ||
		existing.Name != sent.Name ||
		existing.Summary != sent.Summary
}

func (m *catalogManager) rewriteEntry(ctx context.Context, existing, e Entry) (Entry, error) {
	reviewReset := reviewedContentChanged(existing, e)
	e.Visibility, e.ApprovedAt = existing.Visibility, existing.ApprovedAt
	if reviewReset {
		e.Visibility = models.VisibilityPending
		e.ApprovedAt = ""
	}
	if err := m.checkEntry(&e); err != nil {
		return Entry{}, err
	}
	if err := m.checkKind(ctx, e.Doc); err != nil {
		return Entry{}, err
	}

	e.ID = existing.ID
	e.CreatedAt = existing.CreatedAt
	e.UpdatedAt = m.now()
	e.Deleted = false

	if err := m.replaceEntry(ctx, e, reviewReset); err != nil {
		return Entry{}, err
	}
	if reviewReset {
		if err := m.retireLinksFor(ctx, e.Slug, e.UpdatedAt); err != nil {
			return Entry{}, err
		}
	}
	return e, nil
}

func sameDocument(stored, sent string) bool {
	if stored == sent {
		return true
	}
	canonicalStored, ok := canonicalDocument(stored)
	if !ok {
		return false
	}
	canonicalSent, ok := canonicalDocument(sent)
	if !ok {
		return false
	}
	return canonicalStored == canonicalSent
}

func canonicalDocument(doc string) (string, bool) {
	dec := json.NewDecoder(strings.NewReader(doc))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return "", false
	}
	out, err := json.Marshal(v)
	if err != nil {
		return "", false
	}
	return string(out), true
}

func (m *catalogManager) replaceEntry(ctx context.Context, e Entry, reviewReset bool) error {
	row, err := encode(m.st.Object(roleEntry), e)
	if err != nil {
		return err
	}
	if !reviewReset {
		delete(row, "visibility")
		delete(row, "approved_at")
	}
	return m.entries(ctx).Where("id = ?", e.ID).Updates(row).Error
}

func (m *catalogManager) GetEntry(ctx context.Context, slug string) (Entry, error) {
	return m.findEntry(ctx, slug)
}

func (m *catalogManager) OpenEntry(ctx context.Context, slug, key string) (Entry, error) {
	e, err := m.findEntry(ctx, slug)
	if err != nil {
		return Entry{}, err
	}
	if !e.PublishedBy(m.now()) {
		return Entry{}, ErrEntryNotFound
	}

	switch e.Visibility {
	case models.VisibilityPublic:
		return e, nil
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
		return e, nil
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
	return m.rewriteEntry(ctx, existing, e)
}

func (m *catalogManager) SetVisibility(ctx context.Context, slug string, v Visibility) (Entry, error) {
	if err := m.checksField(roleEntry, "visibility", string(v)); err != nil {
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
	if err := m.privates(ctx).Where("slug = ?", slug).Delete(&map[string]any{}).Error; err != nil {
		return err
	}
	return m.retireLinksFor(ctx, slug, now)
}

func (m *catalogManager) ListPublicEntries(ctx context.Context, f ListFilter) ([]Entry, error) {
	f.Visibility = models.VisibilityPublic
	if f.Limit <= 0 || f.Limit > models.MaxPublicEntryLimit {
		f.Limit = models.MaxPublicEntryLimit
	}
	return m.listEntries(ctx, f, m.now())
}

func (m *catalogManager) ListEntries(ctx context.Context, f ListFilter) ([]Entry, error) {
	return m.listEntries(ctx, f, "")
}

func (m *catalogManager) listEntries(ctx context.Context, f ListFilter, publishedBy string) ([]Entry, error) {
	q := m.entries(ctx).Where("deleted = ?", false)
	if publishedBy != "" {
		q = q.Where("(publish_at IS NULL OR publish_at = '' OR publish_at <= ?)", publishedBy)
	}
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

	var rows []map[string]any
	if err := q.Order("name").Find(&rows).Error; err != nil {
		return nil, err
	}

	o := m.st.Object(roleEntry)
	out := make([]Entry, 0, len(rows))
	for _, r := range rows {
		var e Entry
		if err := decode(o, r, &e); err != nil {
			return nil, err
		}
		// A listing is the card. Carrying Doc would put 34 KB per row into
		// every storefront index page.
		e.Doc = ""
		out = append(out, e)
	}
	return out, nil
}

// docPathExpr builds the SQL that reads one path out of the document, in
// the dialect in use.
//
// It defers to [spec.DocPathExpr], which is the same function the migrator
// builds a document index with. That is the whole point of going through it:
// an expression index only applies to a query whose expression matches it
// exactly, so a second copy of this that drifted by a space would leave
// every document filter doing a full scan with nothing reporting it.
//
// Neither dialect takes the path as a bound parameter, which is why
// [models.DocFilter.Valid] restricts it to an alphabet with no quote, brace
// or bracket in it — callers must check Valid before reaching here, and
// ListEntries does.
func docPathExpr(db *gorm.DB, d models.DocFilter) string {
	return spec.DocPathExpr("doc", d.Path, db.Dialector.Name())
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
