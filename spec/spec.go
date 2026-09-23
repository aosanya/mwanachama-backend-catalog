// Package spec is the declared shape of a domain: the objects it stores,
// their fields, and the indexes worth keeping. A module supplies the acts; a
// spec supplies the nouns.
//
// Nothing here knows what a catalog is. The same loader serves an agency
// storefront declaring roles and objectives and a library declaring copies
// and loans, which is the whole reason the objects are data rather than Go
// types.
//
// Identifiers in a spec reach the database as SQL text — a table name cannot
// be a bound parameter — so every name is validated against a strict
// alphabet when the spec loads, and a spec that fails validation is refused
// at startup rather than at the first request.
package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// NamePattern is the only shape an object, field or index name may take:
// lowercase, starting with a letter, words joined by single underscores.
//
// It is strict because these names are interpolated into DDL. Anything a
// caller could use to close an identifier and continue the statement is
// excluded from the alphabet outright rather than escaped.
var NamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// FieldType is what a column holds. The set is closed: a type without an arm
// in [Field.columnType] should fail when the spec loads, not produce a column
// nobody meant.
type FieldType string

const (
	// TypeString is a short identifier or label.
	TypeString FieldType = "string"

	// TypeText is free prose of any length.
	TypeText FieldType = "text"

	// TypeInt is a whole number.
	TypeInt FieldType = "int"

	// TypeBool is true or false.
	TypeBool FieldType = "bool"

	// TypeJSON is a document — jsonb where the database has it.
	TypeJSON FieldType = "json"

	// TypeTimestamp is an RFC 3339 instant, stored as text so every dialect
	// compares it the same way.
	TypeTimestamp FieldType = "timestamp"

	// TypeEnum is a string restricted to Values.
	TypeEnum FieldType = "enum"
)

var fieldTypes = map[FieldType]bool{
	TypeString: true, TypeText: true, TypeInt: true, TypeBool: true,
	TypeJSON: true, TypeTimestamp: true, TypeEnum: true,
}

// Field is one column on one object.
type Field struct {
	Name string    `json:"name"`
	Type FieldType `json:"type"`

	// Primary marks the primary key. An object needs at least one; several
	// make a composite key in declared order.
	Primary bool `json:"primary,omitempty"`

	// Required rejects an empty value at write time.
	Required bool `json:"required,omitempty"`

	// Unique constrains the column across the table.
	Unique bool `json:"unique,omitempty"`

	// Immutable rejects a change after creation.
	Immutable bool `json:"immutable,omitempty"`

	// Default is the value a row takes when the field is absent.
	Default string `json:"default,omitempty"`

	// Values is the permitted set, for an enum.
	Values []string `json:"values,omitempty"`

	// Matches names a validation pattern the module supplies, such as
	// "slug". The spec names it; the module owns what it means.
	Matches string `json:"matches,omitempty"`
}

// Index is one index on one object. Either Fields or Path is set, never
// both: Path indexes a document path, which is how a domain makes its own
// vocabulary fast without the module learning it.
type Index struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields,omitempty"`

	// Path is a dotted path inside a json field, e.g.
	// {"field": "doc", "path": "_template.sector"}.
	Path *PathRef `json:"path,omitempty"`

	Unique bool `json:"unique,omitempty"`

	// NotDeleted restricts the index to rows whose "deleted" column is
	// false. It is a named condition rather than free SQL, because free SQL
	// here would be an injection surface with no upside.
	NotDeleted bool `json:"not_deleted,omitempty"`
}

// PathRef names a path inside a json field.
type PathRef struct {
	Field string `json:"field"`
	Path  string `json:"path"`
}

// Object is one declared thing, and one table.
type Object struct {
	Name string `json:"name"`

	// Table is the table's name within the instance. The instance prefix is
	// added by [Spec.TableFor].
	Table string `json:"table"`

	Fields  []Field `json:"fields"`
	Indexes []Index `json:"indexes,omitempty"`
}

// Spec is one domain's declared shape.
type Spec struct {
	// Module names the Go module whose acts this spec is declared against.
	Module string `json:"module"`

	// Domain names the domain — "agency", "library". It is a label, never
	// read as behaviour.
	Domain string `json:"domain"`

	// Instance prefixes every table, so several domains share a database.
	Instance string `json:"instance"`

	Objects []Object `json:"objects"`
}

// Load reads and validates a spec. A spec that does not validate is refused
// here, which is the only moment at which refusing it is cheap.
func Load(path string) (*Spec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("spec: read %s: %w", path, err)
	}
	return Parse(raw)
}

// Parse validates a spec already in memory.
func Parse(raw []byte) (*Spec, error) {
	var s Spec
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("spec: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

// Validate reports every way the spec is unusable, rather than the first.
func (s *Spec) Validate() error {
	var problems []string
	add := func(format string, a ...any) { problems = append(problems, fmt.Sprintf(format, a...)) }

	if !NamePattern.MatchString(s.Instance) {
		add("instance %q is not a usable name", s.Instance)
	}
	if s.Module == "" {
		add("module is required")
	}
	if len(s.Objects) == 0 {
		add("a spec declares at least one object")
	}

	seenObject := map[string]bool{}
	seenTable := map[string]bool{}
	for _, o := range s.Objects {
		if !NamePattern.MatchString(o.Name) {
			add("object %q is not a usable name", o.Name)
			continue
		}
		if seenObject[o.Name] {
			add("object %q is declared twice", o.Name)
		}
		seenObject[o.Name] = true

		table := o.Table
		if table == "" {
			table = o.Name
		}
		if !NamePattern.MatchString(table) {
			add("object %q: table %q is not a usable name", o.Name, table)
		}
		if seenTable[table] {
			add("table %q is declared by more than one object", table)
		}
		seenTable[table] = true

		problems = append(problems, o.validate()...)
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("spec: %s", strings.Join(problems, "; "))
	}
	return nil
}

func (o Object) validate() []string {
	var problems []string
	add := func(format string, a ...any) { problems = append(problems, fmt.Sprintf(format, a...)) }

	if len(o.Fields) == 0 {
		add("object %q declares no fields", o.Name)
		return problems
	}

	seen := map[string]bool{}
	primaries := 0
	for _, f := range o.Fields {
		if !NamePattern.MatchString(f.Name) {
			add("object %q: field %q is not a usable name", o.Name, f.Name)
			continue
		}
		if seen[f.Name] {
			add("object %q: field %q is declared twice", o.Name, f.Name)
		}
		seen[f.Name] = true

		if !fieldTypes[f.Type] {
			add("object %q: field %q has unknown type %q", o.Name, f.Name, f.Type)
		}
		if f.Type == TypeEnum && len(f.Values) == 0 {
			add("object %q: enum field %q declares no values", o.Name, f.Name)
		}
		if f.Type != TypeEnum && len(f.Values) > 0 {
			add("object %q: field %q declares values but is not an enum", o.Name, f.Name)
		}
		if f.Primary {
			primaries++
		}
		if f.Primary && f.Type == TypeJSON {
			add("object %q: field %q cannot be both a document and a key", o.Name, f.Name)
		}
	}
	if primaries == 0 {
		add("object %q declares no primary key", o.Name)
	}

	seenIndex := map[string]bool{}
	for _, idx := range o.Indexes {
		if !NamePattern.MatchString(idx.Name) {
			add("object %q: index %q is not a usable name", o.Name, idx.Name)
			continue
		}
		if seenIndex[idx.Name] {
			add("object %q: index %q is declared twice", o.Name, idx.Name)
		}
		seenIndex[idx.Name] = true

		switch {
		case idx.Path != nil && len(idx.Fields) > 0:
			add("object %q: index %q names both fields and a path", o.Name, idx.Name)
		case idx.Path != nil:
			if !seen[idx.Path.Field] {
				add("object %q: index %q indexes unknown field %q", o.Name, idx.Name, idx.Path.Field)
			}
			if !DocPathPattern.MatchString(idx.Path.Path) {
				add("object %q: index %q has an unusable path %q", o.Name, idx.Name, idx.Path.Path)
			}
		case len(idx.Fields) > 0:
			for _, f := range idx.Fields {
				if !seen[f] {
					add("object %q: index %q indexes unknown field %q", o.Name, idx.Name, f)
				}
			}
		default:
			add("object %q: index %q indexes nothing", o.Name, idx.Name)
		}

		if idx.NotDeleted && !seen["deleted"] {
			add("object %q: index %q is scoped to undeleted rows but there is no deleted field", o.Name, idx.Name)
		}
	}
	return problems
}

// DocPathPattern is the shape a path inside a document may take. Same
// alphabet and same reason as a name: it reaches SQL as text.
var DocPathPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+)*$`)

// TableFor returns the physical table name for one object.
func (s *Spec) TableFor(o Object) string {
	table := o.Table
	if table == "" {
		table = o.Name
	}
	return s.Instance + "_" + table
}

// Object returns the declared object of that name.
func (s *Spec) Object(name string) (Object, bool) {
	for _, o := range s.Objects {
		if o.Name == name {
			return o, true
		}
	}
	return Object{}, false
}
