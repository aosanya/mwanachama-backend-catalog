package mwanachamacatalog

import (
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/gormstore"
)

// TableNames configures which physical tables a [CatalogManager] reads and
// writes.
type TableNames = gormstore.TableNames

// DefaultTableNames builds the conventional table set for one mounted
// instance of this package.
func DefaultTableNames(instance string) TableNames {
	return gormstore.DefaultTableNames(instance)
}

// Migrate creates or updates the tables t names.
func Migrate(db *gorm.DB, t TableNames) error {
	return gormstore.Migrate(db, t)
}
