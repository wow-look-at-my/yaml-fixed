package yaml

import (
	"bufio"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureWarn redirects Warn into a slice for the duration of a test and
// restores the previous hook afterwards.
func captureWarn(t *testing.T) *[]string {
	t.Helper()
	var msgs []string
	prev := Warn
	Warn = func(m string) { msgs = append(msgs, m) }
	t.Cleanup(func() { Warn = prev })
	return &msgs
}

func TestConsumeJSONObjectSpaceIndented(t *testing.T) {
	msgs := captureWarn(t)
	in := "{\n" +
		"  \"name\": \"Ada\",\n" +
		"  \"port\": 8080,\n" +
		"  \"tags\": [\"x\", \"y\"]\n" +
		"}\n"
	got, err := Parse([]byte(in))
	require.NoError(t, err)
	want := map[string]any{
		"name": "Ada",
		"port": 8080,
		"tags": []any{"x", "y"},
	}
	assert.Equal(t, want, got)
	require.Len(t, *msgs, 1, "space-indented JSON should warn once")
	assert.Contains(t, (*msgs)[0], "JSON")
	assert.Contains(t, (*msgs)[0], "tabs")
}

func TestConsumeJSONArraySpaceIndentedAndNested(t *testing.T) {
	msgs := captureWarn(t)
	in := "[\n" +
		"  1,\n" +
		"  2,\n" +
		"  {\n" +
		"      \"deep\": true\n" +
		"  }\n" +
		"]\n"
	got, err := Parse([]byte(in))
	require.NoError(t, err)
	assert.Equal(t, []any{1, 2, map[string]any{"deep": true}}, got)
	require.Len(t, *msgs, 1)
}

func TestConsumeJSONTabIndentedDoesNotWarn(t *testing.T) {
	msgs := captureWarn(t)
	in := "{\n" +
		"\t\"name\": \"Ada\",\n" +
		"\t\"port\": 8080\n" +
		"}\n"
	got, err := Parse([]byte(in))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"name": "Ada", "port": 8080}, got)
	assert.Empty(t, *msgs, "tab-indented JSON is canonical; no warning")
}

func TestConsumeJSONSingleLineDoesNotWarn(t *testing.T) {
	msgs := captureWarn(t)
	got, err := Parse([]byte(`{"a": 1, "b": [2, 3]}`))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"a": 1, "b": []any{2, 3}}, got)
	assert.Empty(t, *msgs, "single-line JSON has no indentation to flag")
}

func TestConsumeJSONLeadingCommentAndBlankLines(t *testing.T) {
	msgs := captureWarn(t)
	in := "# a leading comment\n" +
		"\n" +
		"{\n" +
		"  \"a\": 1\n" +
		"}\n"
	got, err := Parse([]byte(in))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"a": 1}, got)
	require.Len(t, *msgs, 1)
}

func TestConsumeJSONTrailingComment(t *testing.T) {
	msgs := captureWarn(t)
	in := "{\n" +
		"  \"a\": 1\n" +
		"} # trailing\n"
	got, err := Parse([]byte(in))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"a": 1}, got)
	require.Len(t, *msgs, 1)
}

func TestWarnOncePerFileAcrossDocuments(t *testing.T) {
	msgs := captureWarn(t)
	// Two space-indented JSON documents in one stream: still a single warning.
	in := "{\n  \"a\": 1\n}\n---\n[\n  2,\n  3\n]\n"
	docs, err := ParseAll([]byte(in))
	require.NoError(t, err)
	assert.Equal(t, []any{map[string]any{"a": 1}, []any{2, 3}}, docs)
	assert.Len(t, *msgs, 1, "warning is once per file, not per document or per line")
}

func TestConsumeJSONInvalidStillErrors(t *testing.T) {
	_, err := Parse([]byte("{\n  \"a\": 1\n"))
	require.Error(t, err, "an unterminated flow mapping is still a syntax error")
}

func TestBlockYAMLStillRejectsSpaces(t *testing.T) {
	// The JSON exception must not leak into ordinary block documents.
	msgs := captureWarn(t)
	_, err := Parse([]byte("server:\n  host: localhost\n"))
	require.Error(t, err)
	se, ok := err.(*SyntaxError)
	require.True(t, ok)
	assert.Contains(t, se.Msg, "spaces cannot be used for indentation")
	assert.Empty(t, *msgs, "block YAML never warns; it errors")
}

func TestHasSpaceIndent(t *testing.T) {
	assert.True(t, hasSpaceIndent("  x"))   // pure space indent
	assert.True(t, hasSpaceIndent("\t  x")) // alignment space after a tab
	assert.False(t, hasSpaceIndent("\tx"))  // tab indent only
	assert.False(t, hasSpaceIndent("x"))    // no indent
	assert.False(t, hasSpaceIndent(""))     // empty
	assert.False(t, hasSpaceIndent("\t\t")) // tabs, no content
}

func TestDocumentIsFlow(t *testing.T) {
	mk := func(s string) []physLine {
		var out []physLine
		for i, t := range strings.Split(s, "\n") {
			out = append(out, physLine{text: t, no: i + 1})
		}
		return out
	}
	assert.True(t, documentIsFlow(mk("{a: 1}")))
	assert.True(t, documentIsFlow(mk("[1, 2]")))
	assert.True(t, documentIsFlow(mk("# c\n\n  [1]"))) // skip comment/blank, ignore indent
	assert.False(t, documentIsFlow(mk("a: 1")))
	assert.False(t, documentIsFlow(mk("- a")))
	assert.False(t, documentIsFlow(mk("# only a comment")))
}

// TestWarnDefaultWritesStderr exercises the default Warn implementation (it is
// otherwise replaced by captureWarn everywhere else).
func TestWarnDefaultWritesStderr(t *testing.T) {
	orig := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	Warn("hello from the default hook")
	require.NoError(t, w.Close())

	line, err := bufio.NewReader(r).ReadString('\n')
	require.True(t, err == nil || err == io.EOF)
	assert.Contains(t, line, "yaml: warning:")
	assert.Contains(t, line, "hello from the default hook")
}
