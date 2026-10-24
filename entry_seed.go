package mwanachamacatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const seedsSetting = "seeds"

func (m *catalogManager) SeedEntries(ctx context.Context) ([]string, error) {
	seeds, err := m.entrySeeds(ctx)
	if err != nil {
		return nil, err
	}
	created := []string{}
	for _, s := range seeds {
		_, err := m.CreateEntry(ctx, s.Entry())
		if errors.Is(err, ErrEntryExists) {
			continue
		}
		if err != nil {
			return created, fmt.Errorf("catalog: seed %s: %w", s.Slug, err)
		}
		created = append(created, s.Slug)
	}
	return created, nil
}

func (m *catalogManager) entrySeeds(ctx context.Context) ([]models.EntrySeed, error) {
	if _, declared := m.spec.Setting(seedsSetting); !declared {
		return nil, nil
	}
	effective, err := spec.ResolveStored(m.db.WithContext(ctx), m.spec, seedsSetting)
	if err != nil {
		return nil, fmt.Errorf("catalog: %s: %w", seedsSetting, err)
	}
	return models.ParseEntrySeeds(effective.Value)
}
