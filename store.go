package mwanachamacatalog

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-catalog/spec"
)

// store is the spec-driven store. It knows which table plays which of the
// module's roles and how a domain value becomes a row, and nothing else —
// there are no row structs here, because the columns are declared.
//
// The join between a declared field and a Go field is the field's name, put
// through [columnName]: SubmittedBy is submitted_by. Deliberately not the
// json tag, which is a presentation choice and gets this wrong in exactly
// the place it would hurt — ShareLink.KeyHash is json:"-" so a hash never
// reaches an API response, and a codec reading json tags would quietly stop
// storing it.
type store struct {
	db      *gorm.DB
	spec    *spec.Spec
	objects map[string]spec.Object
}

// The roles this module operates on. A domain calls the objects whatever it
// likes — an agency, a title — and fills these roles with them; every rule
// in this package reaches its table through one of these and never through a
// domain's own noun.
const (
	roleEntry             = "entry"
	roleEntryPrivate      = "entry_private"
	roleShareLink         = "share_link"
	roleSuggestion        = "suggestion"
	roleSuggestionComment = "suggestion_comment"
	roleSuggestionMatch   = "suggestion_match"
	roleRun               = "run"
	roleSetting           = "setting"
	rolePageView          = "page_view"
)

// newStore resolves the object playing each role and checks that every
// declared field has somewhere to live in the Go type that carries it.
//
// models maps a role to a zero value of the type the manager passes for it.
// The check runs once, here, because a column with no field is a value that
// would be dropped on every single write with nothing to show for it.
func newStore(db *gorm.DB, s *spec.Spec, models map[string]any) (*store, error) {
	if db == nil {
		return nil, fmt.Errorf("store: db must not be nil")
	}
	if s == nil {
		return nil, fmt.Errorf("store: spec must not be nil")
	}

	roles := make([]string, 0, len(models))
	for role := range models {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	if err := s.RequireRoles(roles...); err != nil {
		return nil, err
	}

	objects := make(map[string]spec.Object, len(models))
	var problems []string
	for _, role := range roles {
		o, _ := s.ByRole(role)
		objects[role] = o

		declared := map[string]bool{}
		for _, f := range o.Fields {
			declared[f.Name] = true
		}
		carried := columnsOf(reflect.TypeOf(models[role]))

		for name := range declared {
			if !carried[name] {
				problems = append(problems, fmt.Sprintf(
					"%s declares %q, which %s does not carry",
					o.Name, name, reflect.TypeOf(models[role])))
			}
		}
		for name := range carried {
			if !declared[name] {
				problems = append(problems, fmt.Sprintf(
					"%s carries %q, which the object %q does not declare",
					reflect.TypeOf(models[role]), name, o.Name))
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return nil, fmt.Errorf("store: spec and types disagree: %s", strings.Join(problems, "; "))
	}
	return &store{db: db, spec: s, objects: objects}, nil
}

// newID mints a storage key. The row structs minted theirs in a GORM
// BeforeCreate hook, which a map-shaped write never reaches, so the managers
// now do it where the row is built and the id is visible in what they return.
func newID() string { return uuid.NewString() }

// table is the physical table playing one role.
func (st *store) table(role string) string {
	return st.spec.TableFor(st.object(role))
}

// object returns the object playing one role. The role is the module's own
// word, so a missing one is a programming error rather than a bad spec —
// newStore has already refused a spec that does not fill them all.
func (st *store) object(role string) spec.Object {
	o, ok := st.objects[role]
	if !ok {
		panic("store: no object for role " + role)
	}
	return o
}

// columnsOf maps the column each exported field of t would occupy.
func columnsOf(t reflect.Type) map[string]bool {
	out := map[string]bool{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}
		out[columnName(f.Name)] = true
	}
	return out
}

// columnName is the rule that turns a Go field name into a column name:
// SubmittedBy is submitted_by, RunID is run_id, ID is id. A run of capitals
// is one word, so an initialism does not scatter into letters.
func columnName(field string) string {
	var b strings.Builder
	runes := []rune(field)
	for i, r := range runes {
		upper := r >= 'A' && r <= 'Z'
		if upper && i > 0 {
			prevLower := runes[i-1] >= 'a' && runes[i-1] <= 'z' || runes[i-1] >= '0' && runes[i-1] <= '9'
			nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
			if prevLower || nextLower {
				b.WriteByte('_')
			}
		}
		if upper {
			b.WriteRune(r - 'A' + 'a')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// encode turns a domain value into the row the object declares: every
// declared column, always, whether or not the value carries anything for it.
//
// Every column every time is the point. A map missing a key means "leave it
// alone" to an update, which would make clearing a field — an Entry losing
// its ApprovedAt when its document changes — silently impossible.
func encode(o spec.Object, v any) (map[string]any, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("encode %s: want a struct, got %T", o.Name, v)
	}

	byColumn := map[string]reflect.Value{}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		if rt.Field(i).PkgPath != "" {
			continue
		}
		byColumn[columnName(rt.Field(i).Name)] = rv.Field(i)
	}

	row := make(map[string]any, len(o.Fields))
	for _, f := range o.Fields {
		fv, ok := byColumn[f.Name]
		if !ok {
			return nil, fmt.Errorf("encode %s: nothing carries %q", o.Name, f.Name)
		}
		switch f.Type {
		case spec.TypeInt:
			row[f.Name] = fv.Int()
		case spec.TypeBool:
			row[f.Name] = fv.Bool()
		case spec.TypeJSON:
			// An empty document is stored as SQL NULL, so a column that was
			// never written is distinguishable from one holding "".
			if s := fv.String(); s != "" {
				row[f.Name] = s
			} else {
				row[f.Name] = nil
			}
		default:
			row[f.Name] = fv.String()
		}
	}
	return row, nil
}

// decode fills out from one row. out must be a pointer to the struct the
// object was checked against.
func decode(o spec.Object, row map[string]any, out any) error {
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("decode %s: want a pointer to a struct, got %T", o.Name, out)
	}
	rv = rv.Elem()

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.PkgPath != "" {
			continue
		}
		raw, ok := row[columnName(f.Name)]
		if !ok || raw == nil {
			continue // the zero value is what an absent column means
		}
		if err := assign(rv.Field(i), raw); err != nil {
			return fmt.Errorf("decode %s.%s: %w", o.Name, f.Name, err)
		}
	}
	return nil
}

// assign writes one database value into one field, taking the dialects as
// they come: SQLite hands back int64 for a boolean and either a string or a
// byte slice for text, Postgres hands back a byte slice for jsonb.
func assign(field reflect.Value, raw any) error {
	switch field.Kind() {
	case reflect.String:
		switch v := raw.(type) {
		case string:
			field.SetString(v)
		case []byte:
			field.SetString(string(v))
		default:
			return fmt.Errorf("cannot read %T as text", raw)
		}
	case reflect.Bool:
		switch v := raw.(type) {
		case bool:
			field.SetBool(v)
		case int64:
			field.SetBool(v != 0)
		default:
			return fmt.Errorf("cannot read %T as a boolean", raw)
		}
	case reflect.Int, reflect.Int64:
		switch v := raw.(type) {
		case int64:
			field.SetInt(v)
		case int:
			field.SetInt(int64(v))
		case float64:
			field.SetInt(int64(v))
		default:
			return fmt.Errorf("cannot read %T as a number", raw)
		}
	default:
		return fmt.Errorf("no rule for a %s field", field.Kind())
	}
	return nil
}
