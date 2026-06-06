package yaml

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Marshal serialises a Go value as YAML. Mappings are emitted with their
// keys sorted; structs follow field declaration order. Nested values are
// indented with tabs, and sequences of mappings use the expanded (dash-on-its-
// own-line) form so the result parses back through Parse.
func Marshal(v any) ([]byte, error) {
	var b strings.Builder
	if err := writeNode(&b, reflect.ValueOf(v), 0); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

// writeNode emits a value as a standalone block, prefixing every line with
// indent tabs.
func writeNode(b *strings.Builder, v reflect.Value, indent int) error {
	v = indirect(v)
	switch {
	case !v.IsValid():
		writeTabs(b, indent)
		b.WriteString("null\n")
		return nil
	case v.Kind() == reflect.Map:
		return writeMap(b, v, indent)
	case v.Kind() == reflect.Struct:
		return writeStruct(b, v, indent)
	case v.Kind() == reflect.Slice || v.Kind() == reflect.Array:
		return writeSeq(b, v, indent)
	default:
		s, err := formatScalar(v)
		if err != nil {
			return err
		}
		writeTabs(b, indent)
		b.WriteString(s)
		b.WriteByte('\n')
		return nil
	}
}

// entry is one mapping key/value pair, ready to emit.
type entry struct {
	key string
	val reflect.Value
}

// mapEntries returns a map's entries sorted by key.
func mapEntries(v reflect.Value) ([]entry, error) {
	keys := v.MapKeys()
	es := make([]entry, 0, len(keys))
	for _, k := range keys {
		s, err := scalarString(k)
		if err != nil {
			return nil, err
		}
		es = append(es, entry{s, v.MapIndex(k)})
	}
	sort.Slice(es, func(i, j int) bool { return es[i].key < es[j].key })
	return es, nil
}

// structEntries returns a struct's emittable fields in declaration order.
func structEntries(v reflect.Value) []entry {
	t := v.Type()
	var es []entry
	for i := 0; i < t.NumField(); i++ {
		fi, ok := parseField(t.Field(i))
		if !ok {
			continue
		}
		fv := v.Field(fi.index)
		if fi.omitEmpty && isEmptyValue(fv) {
			continue
		}
		es = append(es, entry{fi.name, fv})
	}
	return es
}

// entriesOf returns the mapping entries of v when it is a map or struct.
func entriesOf(v reflect.Value) ([]entry, bool, error) {
	switch v.Kind() {
	case reflect.Map:
		es, err := mapEntries(v)
		return es, true, err
	case reflect.Struct:
		return structEntries(v), true, nil
	}
	return nil, false, nil
}

func writeMap(b *strings.Builder, v reflect.Value, indent int) error {
	es, err := mapEntries(v)
	if err != nil {
		return err
	}
	return writeEntries(b, es, indent)
}

func writeStruct(b *strings.Builder, v reflect.Value, indent int) error {
	return writeEntries(b, structEntries(v), indent)
}

// writeEntries emits a block mapping, one "key:" line per entry, or "{}" when
// there are no entries.
func writeEntries(b *strings.Builder, es []entry, indent int) error {
	if len(es) == 0 {
		writeTabs(b, indent)
		b.WriteString("{}\n")
		return nil
	}
	for _, e := range es {
		writeTabs(b, indent)
		b.WriteString(formatKey(e.key))
		b.WriteByte(':')
		if err := writeAfterColon(b, e.val, indent); err != nil {
			return err
		}
	}
	return nil
}

func writeSeq(b *strings.Builder, v reflect.Value, indent int) error {
	n := v.Len()
	if n == 0 {
		writeTabs(b, indent)
		b.WriteString("[]\n")
		return nil
	}
	for i := 0; i < n; i++ {
		if err := writeSeqItem(b, v.Index(i), indent); err != nil {
			return err
		}
	}
	return nil
}

// writeSeqItem emits one sequence element. A non-empty mapping uses the compact
// aligned form -- the first pair shares the dash line, later pairs align past
// the marker with spaces, so the tab sets the depth and the spaces handle the
// alignment.
//
//	- name: Alice
//	  age: 30
//
// Alignment needs a leading tab to align against, so at the document's left
// margin (depth 0) a mapping item falls back to the dash-on-its-own-line form
// with the body one tab deeper. Both forms parse back identically.
func writeSeqItem(b *strings.Builder, elem reflect.Value, indent int) error {
	e := indirect(elem)
	es, isMapping, err := entriesOf(e)
	if err != nil {
		return err
	}
	if isMapping && len(es) > 0 && indent >= 1 {
		for i, en := range es {
			writeTabs(b, indent)
			if i == 0 {
				b.WriteString("- ")
			} else {
				b.WriteString("  ") // align past the "- " marker
			}
			b.WriteString(formatKey(en.key))
			b.WriteByte(':')
			if err := writeAfterColon(b, en.val, indent); err != nil {
				return err
			}
		}
		return nil
	}
	writeTabs(b, indent)
	b.WriteByte('-')
	return writeAfterDash(b, elem, indent)
}

// writeAfterColon writes the value that follows a "key:" separator, choosing
// inline scalar form or an indented block.
func writeAfterColon(b *strings.Builder, v reflect.Value, indent int) error {
	v = indirect(v)
	if isEmptyContainer(v) {
		b.WriteByte(' ')
		b.WriteString(emptyLiteral(v))
		b.WriteByte('\n')
		return nil
	}
	if isComposite(v) {
		b.WriteByte('\n')
		return writeNode(b, v, indent+1)
	}
	s, err := formatScalar(v)
	if err != nil {
		return err
	}
	b.WriteByte(' ')
	b.WriteString(s)
	b.WriteByte('\n')
	return nil
}

// writeAfterDash writes the value that follows a "-" sequence marker.
func writeAfterDash(b *strings.Builder, v reflect.Value, indent int) error {
	v = indirect(v)
	if isEmptyContainer(v) {
		b.WriteByte(' ')
		b.WriteString(emptyLiteral(v))
		b.WriteByte('\n')
		return nil
	}
	if isComposite(v) {
		b.WriteByte('\n')
		return writeNode(b, v, indent+1)
	}
	s, err := formatScalar(v)
	if err != nil {
		return err
	}
	b.WriteByte(' ')
	b.WriteString(s)
	b.WriteByte('\n')
	return nil
}

func writeTabs(b *strings.Builder, n int) {
	for i := 0; i < n; i++ {
		b.WriteByte('\t')
	}
}

func indirect(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func isComposite(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Map, reflect.Struct, reflect.Slice, reflect.Array:
		return true
	}
	return false
}

func isEmptyContainer(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array:
		return v.Len() == 0
	}
	return false
}

func emptyLiteral(v reflect.Value) string {
	if v.Kind() == reflect.Map {
		return "{}"
	}
	return "[]"
}

// formatScalar renders a scalar reflect.Value as YAML text.
func formatScalar(v reflect.Value) (string, error) {
	if !v.IsValid() {
		return "null", nil
	}
	switch v.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return formatFloat(v.Float(), v.Kind()), nil
	case reflect.String:
		return formatString(v.String()), nil
	default:
		return "", fmt.Errorf("yaml: cannot marshal value of type %s", v.Type())
	}
}

// scalarString renders a value that must serve as a mapping key.
func scalarString(v reflect.Value) (string, error) {
	v = indirect(v)
	if v.Kind() == reflect.String {
		return v.String(), nil
	}
	return formatScalar(v)
}

func formatFloat(f float64, kind reflect.Kind) string {
	switch {
	case math.IsNaN(f):
		return ".nan"
	case math.IsInf(f, 1):
		return ".inf"
	case math.IsInf(f, -1):
		return "-.inf"
	}
	bits := 64
	if kind == reflect.Float32 {
		bits = 32
	}
	s := strconv.FormatFloat(f, 'g', -1, bits)
	// Ensure the result still reads back as a float, not an int.
	if !strings.ContainsAny(s, ".eEnN") {
		s += ".0"
	}
	return s
}

func formatKey(s string) string {
	return formatString(s)
}

func formatString(s string) string {
	if !needsQuote(s) {
		return s
	}
	return quoteDouble(s)
}

// needsQuote reports whether a string must be quoted to round-trip as a string
// scalar (rather than being re-interpreted as another type or breaking syntax).
func needsQuote(s string) bool {
	if s == "" {
		return true
	}
	if _, isStr := resolvePlain(s).(string); !isStr {
		return true // would parse back as null/bool/number
	}
	if s != strings.TrimSpace(s) {
		return true
	}
	if strings.ContainsAny(s, "\n\t\r") {
		return true
	}
	if strings.Contains(s, ": ") || strings.HasSuffix(s, ":") || strings.Contains(s, " #") {
		return true
	}
	switch s[0] {
	case '-', '?', ':', ',', '[', ']', '{', '}', '&', '*', '!', '|', '>', '\'', '"', '%', '@', '`', '#', ' ':
		return true
	}
	return false
}

func quoteDouble(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\x%02x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
