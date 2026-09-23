package gormstore

import (
	"database/sql/driver"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// JSONDoc is a JSON document column. It reads and writes as a Go string so
// the domain types stay free of any storage type, and it declares itself as
// jsonb on Postgres and text everywhere else.
//
// The dialect split is the point. The catalog's reads fold arrays out of a
// document and count fields inside it, which Postgres does in the database
// and only over jsonb; SQLite has no such type at all, and a column
// declared jsonb there lands in NUMERIC affinity, which is not what anyone
// means. The unit tests run on SQLite and the jsonb behaviour is covered by
// the Postgres integration test.
type JSONDoc string

// GormDBDataType picks the column type for the dialect in use.
func (JSONDoc) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Dialector.Name() == "postgres" {
		return "jsonb"
	}
	return "text"
}

// Value writes the document, storing an empty one as SQL NULL so a column
// that was never set is distinguishable from one holding "".
func (d JSONDoc) Value() (driver.Value, error) {
	if d == "" {
		return nil, nil
	}
	return string(d), nil
}

// Scan reads the document back from either a string or a byte slice, which
// is the difference between the SQLite and Postgres drivers.
func (d *JSONDoc) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*d = ""
	case string:
		*d = JSONDoc(v)
	case []byte:
		*d = JSONDoc(v)
	default:
		return fmt.Errorf("JSONDoc.Scan: cannot read %T", src)
	}
	return nil
}
