package yaml

import (
	"reflect"
	"strconv"
	"strings"
)

// Unmarshaler is implemented by a type that decodes itself from the generic
// value its YAML node parses to: nil, bool, int, float64, string, []any, or
// *Map. Unmarshal/UnmarshalStrict call it (via an addressable pointer to the
// destination, so a value receiver on a named type still works) in place of
// the built-in struct/map/slice/scalar decoding for that node -- the same
// dispatch point most YAML libraries call UnmarshalYAML at, adapted to this
// package's generic-value model instead of a node tree.
type Unmarshaler interface {
	UnmarshalYAML(value any) error
}

// Unmarshal parses a single YAML document and stores the result in the
// value pointed to by v. Mappings decode into structs (matching the `yaml` tag
// or the lower-cased field name) or maps; sequences decode into slices; scalars
// decode into the matching Go scalar type or into an interface{}. Unknown
// mapping keys are ignored; use UnmarshalStrict to reject them.
func Unmarshal(data []byte, v any) error {
	return unmarshal(data, v, false)
}

// UnmarshalStrict is Unmarshal, except every struct field decode rejects a
// mapping key with no matching field -- a typo'd or unsupported key is a
// decode error instead of being silently dropped. Strictness is per struct
// field lookup, so it applies at every nesting depth, not just the top level.
func UnmarshalStrict(data []byte, v any) error {
	return unmarshal(data, v, true)
}

func unmarshal(data []byte, v any, strict bool) error {
	generic, err := Parse(data)
	if err != nil {
		return err
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return typeErrorf("Unmarshal requires a non-nil pointer, got %T", v)
	}
	return decode(generic, rv.Elem(), strict)
}

func decode(src any, dst reflect.Value, strict bool) error {
	// Resolve pointers, allocating as needed.
	for dst.Kind() == reflect.Ptr {
		if src == nil {
			if !dst.IsNil() {
				dst.Set(reflect.Zero(dst.Type()))
			}
			return nil
		}
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst = dst.Elem()
	}

	if dst.CanAddr() {
		if u, ok := dst.Addr().Interface().(Unmarshaler); ok {
			return u.UnmarshalYAML(src)
		}
	}

	if src == nil {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	if dst.Kind() == reflect.Interface && dst.NumMethod() == 0 {
		dst.Set(reflect.ValueOf(toPlainInterface(src)))
		return nil
	}

	switch dst.Kind() {
	case reflect.Bool:
		b, ok := src.(bool)
		if !ok {
			return typeErrorf("cannot decode %s into bool", kindOf(src))
		}
		dst.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, ok := asInt(src)
		if !ok {
			return typeErrorf("cannot decode %s into %s", kindOf(src), dst.Kind())
		}
		if dst.OverflowInt(n) {
			return typeErrorf("value %d overflows %s", n, dst.Kind())
		}
		dst.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n, ok := asInt(src)
		if !ok || n < 0 {
			return typeErrorf("cannot decode %s into %s", kindOf(src), dst.Kind())
		}
		if dst.OverflowUint(uint64(n)) {
			return typeErrorf("value %d overflows %s", n, dst.Kind())
		}
		dst.SetUint(uint64(n))
	case reflect.Float32, reflect.Float64:
		f, ok := asFloat(src)
		if !ok {
			return typeErrorf("cannot decode %s into %s", kindOf(src), dst.Kind())
		}
		dst.SetFloat(f)
	case reflect.String:
		s, ok := scalarAsString(src)
		if !ok {
			return typeErrorf("cannot decode %s into string", kindOf(src))
		}
		dst.SetString(s)
	case reflect.Slice:
		return decodeSlice(src, dst, strict)
	case reflect.Array:
		return decodeArray(src, dst, strict)
	case reflect.Map:
		return decodeMap(src, dst, strict)
	case reflect.Struct:
		return decodeStruct(src, dst, strict)
	default:
		return typeErrorf("unsupported target type %s", dst.Type())
	}
	return nil
}

// ToPlain converts a parsed *Map (recursively, including *Map values nested
// inside sequences) into map[string]any, discarding declaration order.
// Marshal sorts a plain map's keys but preserves a *Map's order verbatim, so
// this is the call a canonicalizer (e.g. `yamlfixed-cli fmt`) makes before
// marshaling to get alphabetized output instead of a pass-through reindent.
func ToPlain(src any) any {
	return toPlainInterface(src)
}

// toPlainInterface converts a parsed *Map (recursively, including *Map
// values nested inside sequences) into map[string]any for a caller decoding
// into `any`/interface{} -- such a caller has no field-name schema to
// preserve order against, so it gets the plain, order-free view generic Go
// code already expects from a YAML mapping.
func toPlainInterface(src any) any {
	if list, ok := src.([]any); ok {
		out := make([]any, len(list))
		for i, e := range list {
			out[i] = toPlainInterface(e)
		}
		return out
	}
	m, ok := src.(*Map)
	if !ok {
		return src
	}
	out := make(map[string]any, len(m.Keys))
	for _, k := range m.Keys {
		out[k] = toPlainInterface(m.Values[k])
	}
	return out
}

func decodeSlice(src any, dst reflect.Value, strict bool) error {
	list, ok := src.([]any)
	if !ok {
		return typeErrorf("cannot decode %s into %s", kindOf(src), dst.Type())
	}
	out := reflect.MakeSlice(dst.Type(), len(list), len(list))
	for i, e := range list {
		if err := decode(e, out.Index(i), strict); err != nil {
			return err
		}
	}
	dst.Set(out)
	return nil
}

func decodeArray(src any, dst reflect.Value, strict bool) error {
	list, ok := src.([]any)
	if !ok {
		return typeErrorf("cannot decode %s into %s", kindOf(src), dst.Type())
	}
	n := dst.Len()
	if len(list) < n {
		n = len(list)
	}
	for i := 0; i < n; i++ {
		if err := decode(list[i], dst.Index(i), strict); err != nil {
			return err
		}
	}
	return nil
}

func decodeMap(src any, dst reflect.Value, strict bool) error {
	m, ok := src.(*Map)
	if !ok {
		return typeErrorf("cannot decode %s into %s", kindOf(src), dst.Type())
	}
	if dst.Type().Key().Kind() != reflect.String {
		return typeErrorf("map key type %s is not a string", dst.Type().Key())
	}
	out := reflect.MakeMapWithSize(dst.Type(), m.Len())
	elemType := dst.Type().Elem()
	for _, k := range m.Keys {
		ev := reflect.New(elemType).Elem()
		if err := decode(m.Values[k], ev, strict); err != nil {
			return err
		}
		key := reflect.New(dst.Type().Key()).Elem()
		key.SetString(k)
		out.SetMapIndex(key, ev)
	}
	dst.Set(out)
	return nil
}

func decodeStruct(src any, dst reflect.Value, strict bool) error {
	m, ok := src.(*Map)
	if !ok {
		return typeErrorf("cannot decode %s into struct %s", kindOf(src), dst.Type())
	}
	t := dst.Type()
	byName := map[string]fieldInfo{}
	for i := 0; i < t.NumField(); i++ {
		if fi, ok := parseField(t.Field(i)); ok {
			byName[fi.name] = fi
		}
	}
	for _, k := range m.Keys {
		fi, ok := byName[k]
		if !ok {
			if strict {
				return typeErrorf("unknown field %q for type %s", k, t)
			}
			continue // ignore unknown keys
		}
		if err := decode(m.Values[k], dst.Field(fi.index), strict); err != nil {
			return err
		}
	}
	return nil
}

// scalarAsString coerces any scalar into a string target, the same loose
// coercion gopkg.in/yaml.v3 applies: an unquoted `cmd: true` or `port: 8080`
// decodes into a string field using the scalar's canonical text, exactly as
// if it had been written quoted. Only a mapping or sequence source is
// rejected -- those have no scalar text to coerce.
func scalarAsString(src any) (string, bool) {
	switch v := src.(type) {
	case string:
		return v, true
	case bool:
		return strconv.FormatBool(v), true
	case int:
		return strconv.Itoa(v), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64), true
	}
	return "", false
}

func asInt(src any) (int64, bool) {
	switch n := src.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		if n == float64(int64(n)) {
			return int64(n), true
		}
	}
	return 0, false
}

func asFloat(src any) (float64, bool) {
	switch n := src.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

func kindOf(src any) string {
	switch src.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case int, int64:
		return "int"
	case float64:
		return "float"
	case string:
		return "string"
	case []any:
		return "sequence"
	case *Map:
		return "mapping"
	default:
		return reflect.TypeOf(src).String()
	}
}

// fieldInfo describes a struct field's YAML name and options.
type fieldInfo struct {
	name      string
	index     int
	omitEmpty bool
}

func parseField(sf reflect.StructField) (fieldInfo, bool) {
	if sf.PkgPath != "" {
		return fieldInfo{}, false // unexported
	}
	tag := sf.Tag.Get("yaml")
	if tag == "-" {
		return fieldInfo{}, false
	}
	name := ""
	omit := false
	if tag != "" {
		parts := strings.Split(tag, ",")
		if parts[0] != "" {
			name = parts[0]
		}
		for _, opt := range parts[1:] {
			if opt == "omitempty" {
				omit = true
			}
		}
	}
	if name == "" {
		name = strings.ToLower(sf.Name)
	}
	return fieldInfo{name: name, index: sf.Index[0], omitEmpty: omit}, true
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
