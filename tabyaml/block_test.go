package tabyaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiteralBlockScalar(t *testing.T) {
	got, err := Parse([]byte("text: |\n\tline1\n\tline2\n"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"text": "line1\nline2\n"}, got)
}

func TestLiteralBlockStrip(t *testing.T) {
	got, err := Parse([]byte("text: |-\n\tline1\n\tline2\n"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"text": "line1\nline2"}, got)
}

func TestLiteralBlockKeep(t *testing.T) {
	got, err := Parse([]byte("text: |+\n\tline1\n\n"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"text": "line1\n\n"}, got)
}

func TestBlockScalarKeepsInnerSpaces(t *testing.T) {
	got, err := Parse([]byte("script: |\n\techo hello\n\t    indented with spaces\n"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"script": "echo hello\n    indented with spaces\n"}, got)
}

func TestFoldedBlockScalar(t *testing.T) {
	got, err := Parse([]byte("text: >\n\ta\n\tb\n\n\tc\n"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"text": "a b\nc\n"}, got)
}

func TestEmptyBlockScalar(t *testing.T) {
	got, err := Parse([]byte("text: |\nnext: 1"))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"text": "", "next": 1}, got)
}

func TestBlockScalarInSequence(t *testing.T) {
	got, err := Parse([]byte("- |\n\tone\n\ttwo\n- plain"))
	require.NoError(t, err)
	assert.Equal(t, []any{"one\ntwo\n", "plain"}, got)
}

func TestBlockScalarDoubleChompError(t *testing.T) {
	_, err := Parse([]byte("t: |-+\n\tx\n"))
	assert.Error(t, err)
}
