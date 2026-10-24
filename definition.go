package mwanachamacatalog

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func PutDefinition(db *gorm.DB, s *spec.Spec, values map[string]any) error {
	if v, named := values[kindsSetting]; named {
		if _, err := models.ParseEntryKinds(v); err != nil {
			return fmt.Errorf("catalog: %s: %w", kindsSetting, err)
		}
	}
	return spec.PutDefinitionSettings(db, s, values)
}
