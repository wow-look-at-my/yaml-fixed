package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoundTrip marshals a value to YAML and parses it back, expecting an
// identical generic structure.
func TestRoundTrip(t *testing.T) {
	values := []any{
		map[string]any{
			"name":		"service",
			"port":		8080,
			"enabled":	true,
			"ratio":	0.5,
			"nested": map[string]any{
				"deep": map[string]any{"x": 1, "y": 2},
			},
			"list":		[]any{1, 2, 3},
			"items":	[]any{map[string]any{"k": "v"}, map[string]any{"k": "w"}},
			"empty":	map[string]any{},
			"none":		nil,
			"tricky":	"true",	// must be quoted to survive
		},
	}
	for _, v := range values {
		data, err := Marshal(v)
		require.NoError(t, err)

		// Every indentation byte produced must be a tab, never a space.
		assertTabIndented(t, data)

		got, err := Parse(data)
		require.NoError(t, err, "yaml:\n%s", data)
		assert.Equal(t, v, got, "yaml:\n%s", data)
	}
}

// assertTabIndented verifies that no line begins with a space.
func assertTabIndented(t *testing.T, data []byte) {
	t.Helper()
	atLineStart := true
	for _, b := range data {
		require.False(t, atLineStart && b == ' ')

		atLineStart = b == '\n'
	}
}
