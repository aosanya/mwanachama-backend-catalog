package mwanachamacatalog

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/gormstore"
	"github.com/aosanya/mwanachama-backend-catalog/models"
)

func (m *catalogManager) runs(ctx context.Context) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.tables.MiningRuns)
}

func (m *catalogManager) settings(ctx context.Context) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.tables.Settings)
}

func (m *catalogManager) views(ctx context.Context) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.tables.PageViews)
}

func (m *catalogManager) RecordRun(ctx context.Context, r MiningRun) (MiningRun, error) {
	if !r.Validate() {
		return MiningRun{}, ErrInvalidRun
	}
	r.ID = ""
	r.CreatedAt = m.now()
	row := gormstore.MiningRunToRow(r)
	if err := m.runs(ctx).Create(&row).Error; err != nil {
		return MiningRun{}, err
	}
	return gormstore.RowToMiningRun(row), nil
}

func (m *catalogManager) ListRuns(ctx context.Context, limit int) ([]MiningRun, error) {
	q := m.runs(ctx).Order("created_at DESC, id DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []gormstore.MiningRunRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]MiningRun, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.RowToMiningRun(r))
	}
	return out, nil
}

func (m *catalogManager) PutSetting(ctx context.Context, key, value string) (Setting, error) {
	if key == "" {
		return Setting{}, ErrInvalidSetting
	}
	s := models.Setting{Key: key, Value: value, UpdatedAt: m.now()}
	row := gormstore.SettingToRow(s)

	var n int64
	if err := m.settings(ctx).Where("key = ?", key).Count(&n).Error; err != nil {
		return Setting{}, err
	}
	if n > 0 {
		if err := m.settings(ctx).Where("key = ?", key).
			Updates(map[string]any{"value": value, "updated_at": s.UpdatedAt}).Error; err != nil {
			return Setting{}, err
		}
		return s, nil
	}
	if err := m.settings(ctx).Create(&row).Error; err != nil {
		return Setting{}, err
	}
	return gormstore.RowToSetting(row), nil
}

func (m *catalogManager) GetSetting(ctx context.Context, key string) (Setting, error) {
	var row gormstore.SettingRow
	err := m.settings(ctx).Where("key = ?", key).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Setting{}, ErrSettingNotFound
	}
	if err != nil {
		return Setting{}, err
	}
	return gormstore.RowToSetting(row), nil
}

func (m *catalogManager) RecordPageView(ctx context.Context, page PageKind, slug string) error {
	if page == models.PageKindIndex {
		slug = ""
	}
	v := models.PageView{Page: page, Slug: slug}
	if !v.Validate() {
		return ErrInvalidPageView
	}
	// An Entry view counts only against a slug that exists, so a caller
	// cannot grow this table by inventing slugs.
	if page == models.PageKindEntry {
		if _, err := m.findEntry(ctx, slug); err != nil {
			return err
		}
	}

	res := m.views(ctx).Where("page = ? AND slug = ?", string(page), slug).
		UpdateColumn("views", gorm.Expr("views + 1"))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		return nil
	}
	row := gormstore.PageViewToRow(models.PageView{Page: page, Slug: slug, Views: 1})
	if err := m.views(ctx).Create(&row).Error; err != nil {
		// Another caller created the row between the update and here; the
		// increment is theirs to have made, so retry it rather than fail.
		if isUniqueViolation(err) {
			return m.views(ctx).Where("page = ? AND slug = ?", string(page), slug).
				UpdateColumn("views", gorm.Expr("views + 1")).Error
		}
		return err
	}
	return nil
}

func (m *catalogManager) ListPageViews(ctx context.Context) ([]PageView, error) {
	var rows []gormstore.PageViewRow
	if err := m.views(ctx).Order("views DESC, page, slug").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]PageView, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.RowToPageView(r))
	}
	return out, nil
}
