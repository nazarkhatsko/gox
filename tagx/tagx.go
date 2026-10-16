// Package tagx binds a map of options (a connection's config, an action's
// with) to a struct, as its field tags describe:
//
//	type Config struct {
//		ProjectID  string `key:"project_id" description:"GCP project" validate:"required"`
//		DatabaseID string `key:"database_id" default:"(default)"`
//		KeyFile    string `key:"credentials_file" validate:"required"`
//	}
//
//	cfg, err := tagx.Decode[Config](raw)
//
// Tags:
//
//   - key: the option's name in the map; a field without it is not bound
//   - type: string, int, float, bool, map, list, any or object (a nested
//     struct, whose own fields are tagged too); checked against the
//     field's Go type, and inferred from it when left out
//   - description: what the option is, for docs
//   - default: the value when the option is absent (not for map, list, any)
//   - validate: comma-separated rules of github.com/go-playground/validator
//     (min=1, max=10, oneof=a b, url, email, ...), checked on the decoded
//     value when the option is set. "required" is tagx's own: the option
//     must be set — in the map or by default — so an explicit 0 or false
//     counts. Cross-field rules (required_if, eqfield, ...) are not
//     supported: each option is checked alone.
//
// Fields lists the options for validation and docs.
package tagx

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Field is one option of a struct, as its tags describe it.
type Field struct {
	Key         string
	Type        string
	Description string
	Default     string
	Required    bool
	// Rules are the validator rules besides required ("" for none).
	Rules string
	// Fields are the fields of a nested struct (Type "object").
	Fields []Field

	index []int
}

// validate checks Field.Rules.
var validate = validator.New()

// Fields returns the options of T, a struct, in field order. A malformed
// tag is an error: tags are code, so it is a bug in T.
func Fields[T any]() ([]Field, error) {
	return fieldsOf(reflect.TypeFor[T]())
}

// MustFields is Fields that panics on a malformed tag, for package-level
// variables.
func MustFields[T any]() []Field {
	fields, err := Fields[T]()
	if err != nil {
		panic(err)
	}

	return fields
}

// Decode returns a T filled from raw: each bound field from its key, else
// its default. A missing required option, a value of the wrong type or one
// failing a rule is an error naming the key. Keys raw has but T doesn't
// bind are ignored.
func Decode[T any](raw map[string]any) (T, error) {
	var dst T

	fields, err := Fields[T]()
	if err != nil {
		return dst, err
	}

	if err := decodeStruct(reflect.ValueOf(&dst).Elem(), fields, raw); err != nil {
		return dst, err
	}

	return dst, nil
}

// decodeStruct fills the fields of v, a struct, from raw.
func decodeStruct(v reflect.Value, fields []Field, raw map[string]any) error {
	for _, f := range fields {
		value, ok := lookup(f, raw)
		if !ok {
			if f.Required {
				return fmt.Errorf("%s is required", f.Key)
			}

			continue
		}

		field := v.FieldByIndex(f.index)
		if err := set(field, f, value); err != nil {
			return fmt.Errorf("%s: %w", f.Key, err)
		}

		if err := check(f, field.Interface()); err != nil {
			return err
		}
	}

	return nil
}

// Encode turns v, a struct, into a map by its tags: each field with a key
// under that key, a nested struct as a map. A nil map, slice, pointer or
// interface becomes an untyped nil, so it reads as null (None in
// mutate_on scripts) rather than an empty collection. Tags other than key
// are not used.
func Encode[T any](v T) (map[string]any, error) {
	fields, err := Fields[T]()
	if err != nil {
		return nil, err
	}

	return encodeStruct(reflect.ValueOf(v), fields), nil
}

func encodeStruct(v reflect.Value, fields []Field) map[string]any {
	out := make(map[string]any, len(fields))
	for _, f := range fields {
		out[f.Key] = encodeValue(v.FieldByIndex(f.index), f.Fields)
	}

	return out
}

func encodeValue(v reflect.Value, nested []Field) any {
	switch v.Kind() {
	case reflect.Struct:
		return encodeStruct(v, nested)
	case reflect.Map, reflect.Slice, reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil
		}
	}

	return v.Interface()
}

// check runs f's validator rules on value.
func check(f Field, value any) error {
	if f.Rules == "" {
		return nil
	}

	err := validate.Var(value, f.Rules)

	var errs validator.ValidationErrors
	if errors.As(err, &errs) && len(errs) > 0 {
		rule := errs[0].Tag()
		if p := errs[0].Param(); p != "" {
			rule += "=" + p
		}

		if f.Key == "" {
			return fmt.Errorf("%v fails %s", value, rule)
		}

		return fmt.Errorf("%s: %v fails %s", f.Key, value, rule)
	}

	if err != nil {
		return fmt.Errorf("validate %q: %w", f.Rules, err)
	}

	return nil
}

// lookup returns the raw value of f: from raw, or its default (as a
// string, converted by set).
func lookup(f Field, raw map[string]any) (any, bool) {
	if value, ok := raw[f.Key]; ok && value != nil {
		return value, true
	}

	if f.Default != "" {
		return f.Default, true
	}

	return nil, false
}

func fieldsOf(t reflect.Type) ([]Field, error) {
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("tagx: %s is not a struct", t)
	}

	var fields []Field

	for i := range t.NumField() {
		sf := t.Field(i)

		key, ok := sf.Tag.Lookup("key")
		if !ok {
			continue
		}

		f, err := fieldOf(sf, key)
		if err != nil {
			return nil, fmt.Errorf("tagx: %s.%s: %w", t, sf.Name, err)
		}

		fields = append(fields, f)
	}

	return fields, nil
}

func fieldOf(sf reflect.StructField, key string) (Field, error) {
	if !sf.IsExported() {
		return Field{}, fmt.Errorf("field is not exported")
	}

	if key == "" {
		return Field{}, fmt.Errorf("key is empty")
	}

	goType, ok := typeOf(sf.Type)
	if !ok {
		return Field{}, fmt.Errorf("unsupported type %s", sf.Type)
	}

	if goType == "object" {
		nested, err := fieldsOf(sf.Type)
		if err != nil {
			return Field{}, err
		}

		return Field{Key: key, Type: goType, Description: sf.Tag.Get("description"), Fields: nested, index: sf.Index}, nil
	}

	f := Field{
		Key:         key,
		Type:        goType,
		Description: sf.Tag.Get("description"),
		Default:     sf.Tag.Get("default"),
		index:       sf.Index,
	}

	if t, ok := sf.Tag.Lookup("type"); ok && t != goType {
		return Field{}, fmt.Errorf("type %q doesn't match the field's %s (%s)", t, sf.Type, goType)
	}

	if f.Default != "" {
		if err := checkDefault(sf.Type, f.Default); err != nil {
			return Field{}, err
		}
	}

	if rules, ok := sf.Tag.Lookup("validate"); ok {
		var rest []string

		for _, rule := range strings.Split(rules, ",") {
			if rule = strings.TrimSpace(rule); rule == "required" {
				f.Required = true
			} else if rule != "" {
				rest = append(rest, rule)
			}
		}

		f.Rules = strings.Join(rest, ",")

		if err := checkRules(sf.Type, f.Rules); err != nil {
			return Field{}, err
		}
	}

	return f, nil
}

// checkRules reports rules validator doesn't know, by running them once on
// a zero value: validator panics on an undefined rule.
func checkRules(t reflect.Type, rules string) (err error) {
	if rules == "" {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("validate %q: %v", rules, r)
		}
	}()

	_ = validate.Var(reflect.Zero(t).Interface(), rules)

	return nil
}

// typeOf names the option type of a Go type.
func typeOf(t reflect.Type) (string, bool) {
	switch t.Kind() {
	case reflect.String:
		return "string", true
	case reflect.Int, reflect.Int64, reflect.Int32:
		return "int", true
	case reflect.Float64, reflect.Float32:
		return "float", true
	case reflect.Bool:
		return "bool", true
	case reflect.Map:
		return "map", t.Key().Kind() == reflect.String
	case reflect.Slice:
		return "list", true
	case reflect.Interface:
		return "any", t.NumMethod() == 0
	case reflect.Struct:
		return "object", true
	default:
		return "", false
	}
}

func checkDefault(t reflect.Type, def string) error {
	if err := set(reflect.New(t).Elem(), Field{}, def); err != nil {
		return fmt.Errorf("default %q: %w", def, err)
	}

	return nil
}

// set stores value in v, converting what YAML produces: numbers across int
// and float when lossless, and strings (defaults) to numbers and bools.
func set(v reflect.Value, f Field, value any) error {
	switch v.Kind() {
	case reflect.String:
		s, ok := value.(string)
		if !ok {
			return mismatch("string", value)
		}

		v.SetString(s)
	case reflect.Int, reflect.Int64, reflect.Int32:
		n, err := toInt(value)
		if err != nil {
			return err
		}

		if v.OverflowInt(n) {
			return fmt.Errorf("%d is out of range", n)
		}

		v.SetInt(n)
	case reflect.Float64, reflect.Float32:
		x, err := toFloat(value)
		if err != nil {
			return err
		}

		v.SetFloat(x)
	case reflect.Bool:
		b, err := toBool(value)
		if err != nil {
			return err
		}

		v.SetBool(b)
	case reflect.Struct:
		raw, ok := value.(map[string]any)
		if !ok {
			return mismatch("object", value)
		}

		nested, err := fieldsOf(v.Type())
		if err != nil {
			return err
		}

		return decodeStruct(v, nested, raw)
	case reflect.Map, reflect.Slice, reflect.Interface:
		rv := reflect.ValueOf(value)
		if !rv.Type().AssignableTo(v.Type()) {
			if v.Kind() == reflect.Slice && rv.Kind() == reflect.Slice {
				return setList(v, rv)
			}

			return mismatch(typeName(v.Type()), value)
		}

		v.Set(rv)
	}

	return nil
}

// setList copies a []any into a typed slice ([]string, ...).
func setList(v, rv reflect.Value) error {
	out := reflect.MakeSlice(v.Type(), rv.Len(), rv.Len())
	for i := range rv.Len() {
		if err := set(out.Index(i), Field{}, rv.Index(i).Interface()); err != nil {
			return fmt.Errorf("[%d]: %w", i, err)
		}
	}

	v.Set(out)

	return nil
}

func toInt(value any) (int64, error) {
	switch n := value.(type) {
	case int:
		return int64(n), nil
	case int64:
		return n, nil
	case float64:
		if n != float64(int64(n)) {
			return 0, fmt.Errorf("want an int, got %v", n)
		}

		return int64(n), nil
	case string:
		i, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("want an int, got %q", n)
		}

		return i, nil
	default:
		return 0, mismatch("int", value)
	}
}

func toFloat(value any) (float64, error) {
	switch n := value.(type) {
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case string:
		x, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0, fmt.Errorf("want a float, got %q", n)
		}

		return x, nil
	default:
		return 0, mismatch("float", value)
	}
}

func toBool(value any) (bool, error) {
	switch b := value.(type) {
	case bool:
		return b, nil
	case string:
		x, err := strconv.ParseBool(b)
		if err != nil {
			return false, fmt.Errorf("want a bool, got %q", b)
		}

		return x, nil
	default:
		return false, mismatch("bool", value)
	}
}

func typeName(t reflect.Type) string {
	name, _ := typeOf(t)

	return name
}

func mismatch(want string, got any) error {
	return fmt.Errorf("want %s, got %T", want, got)
}

// goTypes are the Go types Check converts a value to, by option type.
var goTypes = map[string]reflect.Type{
	"string": reflect.TypeFor[string](),
	"int":    reflect.TypeFor[int64](),
	"float":  reflect.TypeFor[float64](),
	"bool":   reflect.TypeFor[bool](),
	"map":    reflect.TypeFor[map[string]any](),
	"list":   reflect.TypeFor[[]any](),
	"any":    reflect.TypeFor[any](),
}

// Check reports whether value suits an option of type typ (as in Field.Type)
// and passes rules (as in Field.Rules), the way Decode would decode and
// check it. An unknown or empty typ checks nothing.
func Check(typ, rules string, value any) error {
	t, ok := goTypes[typ]
	if !ok || value == nil {
		return nil
	}

	v := reflect.New(t).Elem()
	if err := set(v, Field{}, value); err != nil {
		return err
	}

	return check(Field{Rules: rules, Key: ""}, v.Interface())
}
