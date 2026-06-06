package yaml

import (
	"reflect"
	"strings"
)

// Unmarshal parses a single YAML document and stores the result in the
// value pointed to by v. Mappings decode into structs (matching the `yaml` tag
// or the lower-cased field name) or maps; sequences decode into slices; scalars
// decode into the matching Go scalar type or into an interface{}.
func Unmarshal(data []byte, v any) error {
	generic, err := Parse(data)
	if err != nil {
		return err
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return typeErrorf("Unmarshal requires a non-nil pointer, got %T", v)
	}
	return decode(generic, rv.Elem())
}

func decode(src any, dst reflect.Value) error {
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

	if src == nil {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	if dst.Kind() == reflect.Interface && dst.NumMethod() == 0 {
		dst.Set(reflect.ValueOf(src))
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
		s, ok := src.(string)
		if !ok {
			return typeErrorf("cannot decode %s into string", kindOf(src))
		}
		dst.SetString(s)
	case reflect.Slice:
		return decodeSlice(src, dst)
	case reflect.Array:
		return decodeArray(src, dst)
	case reflect.Map:
		return decodeMap(src, dst)
	case reflect.Struct:
		return decodeStruct(src, dst)
	default:
		return typeErrorf("unsupported target type %s", dst.Type())
	}
	return nil
}

func decodeSlice(src any, dst reflect.Value) error {
	list, ok := src.([]any)
	if !ok {
		return typeErrorf("cannot decode %s into %s", kindOf(src), dst.Type())
	}
	out := reflect.MakeSlice(dst.Type(), len(list), len(list))
	for i, e := range list {
		if err := decode(e, out.Index(i)); err != nil {
			return err
		}
	}
	dst.Set(out)
	return nil
}

func decodeArray(src any, dst reflect.Value) error {
	list, ok := src.([]any)
	if !ok {
		return typeErrorf("cannot decode %s into %s", kindOf(src), dst.Type())
	}
	n := dst.Len()
	if len(list) < n {
		n = len(list)
	}
	for i := 0; i < n; i++ {
		if err := decode(list[i], dst.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

func decodeMap(src any, dst reflect.Value) error {
	m, ok := src.(map[string]any)
	if !ok {
		return typeErrorf("cannot decode %s into %s", kindOf(src), dst.Type())
	}
	if dst.Type().Key().Kind() != reflect.String {
		return typeErrorf("map key type %s is not a string", dst.Type().Key())
	}
	out := reflect.MakeMapWithSize(dst.Type(), len(m))
	elemType := dst.Type().Elem()
	for k, v := range m {
		ev := reflect.New(elemType).Elem()
		if err := decode(v, ev); err != nil {
			return err
		}
		key := reflect.New(dst.Type().Key()).Elem()
		key.SetString(k)
		out.SetMapIndex(key, ev)
	}
	dst.Set(out)
	return nil
}

func decodeStruct(src any, dst reflect.Value) error {
	m, ok := src.(map[string]any)
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
	for k, v := range m {
		fi, ok := byName[k]
		if !ok {
			continue // ignore unknown keys
		}
		if err := decode(v, dst.Field(fi.index)); err != nil {
			return err
		}
	}
	return nil
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
	case map[string]any:
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
