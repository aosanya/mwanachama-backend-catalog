package mwanachamacatalog

import "context"

type AreaDirectory interface {
	AreasOf(ctx context.Context, personIDs []string) (map[string]string, error)
	Areas(ctx context.Context) ([]AreaPopulation, error)
}

func (m *catalogManager) UseAreaDirectory(directory AreaDirectory) {
	m.directory = directory
}

func (m *catalogManager) TallyBoardItemsByArea(ctx context.Context, boardItemID string) ([]BoardItemAreaTally, error) {
	return nil, ErrNotBuilt
}
