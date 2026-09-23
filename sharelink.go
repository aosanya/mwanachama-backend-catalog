package mwanachamacatalog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/gormstore"
	"github.com/aosanya/mwanachama-backend-catalog/models"
)

func (m *catalogManager) links(ctx context.Context) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.tables.ShareLinks)
}

func (m *catalogManager) IssueShareLink(ctx context.Context, slug, label, by string) (ShareLink, string, error) {
	entry, err := m.findEntry(ctx, slug)
	if err != nil {
		return ShareLink{}, "", err
	}
	if models.Visibility(entry.Visibility) != models.VisibilityPrivate {
		return ShareLink{}, "", ErrNotPrivate
	}

	raw, err := newShareKey()
	if err != nil {
		return ShareLink{}, "", err
	}

	link := models.ShareLink{
		Slug:      slug,
		KeyHash:   models.HashKey(raw),
		Label:     label,
		Status:    models.LinkStatusActive,
		CreatedBy: by,
		CreatedAt: m.now(),
	}
	row := gormstore.ShareLinkToRow(link)
	if err := m.links(ctx).Create(&row).Error; err != nil {
		return ShareLink{}, "", err
	}
	return gormstore.RowToShareLink(row), raw, nil
}

func (m *catalogManager) RetireShareLink(ctx context.Context, id string) (ShareLink, error) {
	var row gormstore.ShareLinkRow
	err := m.links(ctx).Where("id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ShareLink{}, ErrShareLinkNotFound
	}
	if err != nil {
		return ShareLink{}, err
	}
	if models.LinkStatus(row.Status) == models.LinkStatusRetired {
		return gormstore.RowToShareLink(row), nil
	}

	now := m.now()
	if err := m.links(ctx).Where("id = ?", id).Updates(map[string]any{
		"status": string(models.LinkStatusRetired), "retired_at": now,
	}).Error; err != nil {
		return ShareLink{}, err
	}
	row.Status, row.RetiredAt = string(models.LinkStatusRetired), now
	return gormstore.RowToShareLink(row), nil
}

func (m *catalogManager) ListShareLinks(ctx context.Context, slug string) ([]ShareLink, error) {
	if _, err := m.findEntry(ctx, slug); err != nil {
		return nil, err
	}
	var rows []gormstore.ShareLinkRow
	if err := m.links(ctx).Where("slug = ?", slug).
		Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ShareLink, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.RowToShareLink(r))
	}
	return out, nil
}

// keyOpens reports whether raw is a live key for slug. It matches on the
// hash, so the raw key never has to be stored to be checked.
func (m *catalogManager) keyOpens(ctx context.Context, slug, raw string) (bool, error) {
	var n int64
	err := m.links(ctx).
		Where("slug = ? AND key_hash = ? AND status = ?",
			slug, models.HashKey(raw), string(models.LinkStatusActive)).
		Count(&n).Error
	return n > 0, err
}

// retireLinksFor retires every live key for slug.
func (m *catalogManager) retireLinksFor(ctx context.Context, slug, now string) error {
	return m.links(ctx).
		Where("slug = ? AND status = ?", slug, string(models.LinkStatusActive)).
		Updates(map[string]any{
			"status": string(models.LinkStatusRetired), "retired_at": now,
		}).Error
}

// newShareKey mints 32 bytes of randomness, hex-encoded — the same 64-char
// shape the mining token already uses.
func newShareKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("newShareKey: %w", err)
	}
	return hex.EncodeToString(b), nil
}
