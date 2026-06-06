package tabyaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseScalars(t *testing.T) {
	cases := []struct {
		in   string
		want any
	}{
		{"42", 42},
		{"-7", -7},
		{"3.14", 3.14},
		{"true", true},
		{"False", false},
		{"null", nil},
		{"~", nil},
		{"hello world", "hello world"},
		{"0x1F", 31},
		{"0o17", 15},
		{`"quoted"`, "quoted"},
		{"'sing'", "sing"},
		{"", nil},
	}
	for _, c := range cases {
		got, err := Parse([]byte(c.in))
		require.NoError(t, err, "input %q", c.in)
		assert.Equal(t, c.want, got, "input %q", c.in)
	}
}

func TestParseMapping(t *testing.T) {
	got, err := Parse([]byte("name: Ada\nage: 36\nactive: true"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"name": "Ada", "age": 36, "active": true}, got)
}

func TestParseNestedMappingTabs(t *testing.T) {
	in := "server:\n\thost: localhost\n\tport: 8080\n\ttls:\n\t\tenabled: true"
	got, err := Parse([]byte(in))
	require.NoError(t, err)
	want := map[string]any{
		"server": map[string]any{
			"host": "localhost",
			"port": 8080,
			"tls":  map[string]any{"enabled": true},
		},
	}
	assert.Equal(t, want, got)
}

func TestParseSequenceScalars(t *testing.T) {
	got, err := Parse([]byte("- apple\n- banana\n- 3"))
	require.NoError(t, err)
	assert.Equal(t, []any{"apple", "banana", 3}, got)
}

func TestParseSequenceOfMappingsExpanded(t *testing.T) {
	in := "people:\n\t-\n\t\tname: Alice\n\t\tage: 30\n\t-\n\t\tname: Bob\n\t\tage: 25"
	got, err := Parse([]byte(in))
	require.NoError(t, err)
	want := map[string]any{
		"people": []any{
			map[string]any{"name": "Alice", "age": 30},
			map[string]any{"name": "Bob", "age": 25},
		},
	}
	assert.Equal(t, want, got)
}

func TestParseSequenceOfMappingsAligned(t *testing.T) {
	// Dash on its own line; mapping body at the SAME tab depth, aligned past
	// the marker with spaces. Tabs for indentation, spaces for alignment.
	in := "people:\n\t-\n\t  name: Alice\n\t  age: 30\n\t-\n\t  name: Bob\n\t  age: 25\n"
	got, err := Parse([]byte(in))
	require.NoError(t, err)
	want := map[string]any{
		"people": []any{
			map[string]any{"name": "Alice", "age": 30},
			map[string]any{"name": "Bob", "age": 25},
		},
	}
	assert.Equal(t, want, got)
}

func TestParseSequenceInlineFirstKeyAligned(t *testing.T) {
	// The canonical pretty form: first key shares the dash line, the rest align.
	in := "people:\n\t- name: Alice\n\t  age: 30\n\t- name: Bob\n\t  age: 25\n"
	got, err := Parse([]byte(in))
	require.NoError(t, err)
	want := map[string]any{
		"people": []any{
			map[string]any{"name": "Alice", "age": 30},
			map[string]any{"name": "Bob", "age": 25},
		},
	}
	assert.Equal(t, want, got)
}

func TestAlignmentSpacesAreNotDepth(t *testing.T) {
	// age aligned with name (same tab depth) => sibling key of the item mapping.
	got, err := Parse([]byte("items:\n\t- key: v\n\t  other: w\n"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"items": []any{
		map[string]any{"key": "v", "other": "w"},
	}}, got)

	// sub indented one more TAB than key => child of key; other stays a sibling.
	got, err = Parse([]byte("items:\n\t- key:\n\t\tsub: 1\n\t  other: w\n"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"items": []any{
		map[string]any{"key": map[string]any{"sub": 1}, "other": "w"},
	}}, got)
}

func TestTabAfterSpacesRejected(t *testing.T) {
	_, err := Parse([]byte("a:\n\t \tb: 1\n"))
	require.Error(t, err)
	se, ok := err.(*SyntaxError)
	require.True(t, ok)
	assert.Contains(t, se.Msg, "tab after spaces")
}

func TestParseInlinePairItem(t *testing.T) {
	got, err := Parse([]byte("- key: value\n- 7"))
	require.NoError(t, err)
	assert.Equal(t, []any{map[string]any{"key": "value"}, 7}, got)
}

func TestParseNestedSequenceUnderKey(t *testing.T) {
	got, err := Parse([]byte("items:\n\t- a\n\t- b"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"items": []any{"a", "b"}}, got)
}

func TestParseNullValues(t *testing.T) {
	got, err := Parse([]byte("a:\nb: ~\nc: null"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"a": nil, "b": nil, "c": nil}, got)
}

func TestParseComments(t *testing.T) {
	in := "# leading comment\nname: Ada # trailing\n\t# indented comment is fine if tabbed\nage: 1"
	got, err := Parse([]byte(in))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"name": "Ada", "age": 1}, got)
}

func TestBlankLinesIgnored(t *testing.T) {
	got, err := Parse([]byte("a: 1\n\n\nb: 2\n   \n"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"a": 1, "b": 2}, got)
}

// --- The headline behaviour: spaces are rejected as indentation. ---

func TestSpaceIndentationRejected(t *testing.T) {
	cases := []string{
		"server:\n  host: localhost",   // two spaces, no tab
		"server:\n    host: localhost", // four spaces, no tab
		"list:\n  - item",              // space-indented sequence
		" key: value",                  // single leading space at root
	}
	for _, in := range cases {
		_, err := Parse([]byte(in))
		require.Error(t, err, "input %q", in)
		se, ok := err.(*SyntaxError)
		require.True(t, ok, "input %q error type %T", in, err)
		assert.Contains(t, se.Msg, "spaces cannot be used for indentation", "input %q", in)
	}
}

func TestSpacesAllowedInValues(t *testing.T) {
	// Spaces are perfectly fine everywhere that is not indentation.
	in := "greeting: hello there friend\nlist: [1, 2, 3]\npair: {a: 1, b: 2}"
	got, err := Parse([]byte(in))
	require.NoError(t, err)
	want := map[string]any{
		"greeting": "hello there friend",
		"list":     []any{1, 2, 3},
		"pair":     map[string]any{"a": 1, "b": 2},
	}
	assert.Equal(t, want, got)
}

func TestErrorLineNumbers(t *testing.T) {
	_, err := Parse([]byte("a: 1\nb: 2\n  c: 3")) // error on line 3
	se, ok := err.(*SyntaxError)
	require.True(t, ok, "want *SyntaxError, got %T (%v)", err, err)
	assert.Equal(t, 3, se.Line)
	assert.Equal(t, 1, se.Col)
}

func TestParseMultiDocument(t *testing.T) {
	docs, err := ParseAll([]byte("---\na: 1\n---\nb: 2\n...\n"))
	require.NoError(t, err)
	want := []any{map[string]any{"a": 1}, map[string]any{"b": 2}}
	assert.Equal(t, want, docs)
}

func TestParseInlineDocumentMarker(t *testing.T) {
	docs, err := ParseAll([]byte("--- 42\n--- hello"))
	require.NoError(t, err)
	assert.Equal(t, []any{42, "hello"}, docs)
}

func TestParseRejectsMultiDocInParse(t *testing.T) {
	_, err := Parse([]byte("---\na: 1\n---\nb: 2"))
	assert.Error(t, err)
}

func TestDuplicateKeyRejected(t *testing.T) {
	_, err := Parse([]byte("a: 1\na: 2"))
	assert.Error(t, err)
}

func TestMappingThenSequenceMismatch(t *testing.T) {
	_, err := Parse([]byte("a: 1\n- 2"))
	assert.Error(t, err)
}

func TestDirectivesSkipped(t *testing.T) {
	got, err := Parse([]byte("%YAML 1.2\n---\nok: true"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"ok": true}, got)
}

func TestCRLFNormalised(t *testing.T) {
	got, err := Parse([]byte("a: 1\r\nb:\r\n\tc: 2\r\n"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"a": 1, "b": map[string]any{"c": 2}}, got)
}

func TestQuotedKeyWithColon(t *testing.T) {
	got, err := Parse([]byte(`"a: b": value`))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"a: b": "value"}, got)
}

func TestValueWithColon(t *testing.T) {
	got, err := Parse([]byte("time: 12:30:00"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"time": "12:30:00"}, got)
}
