package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFlowSequence(t *testing.T) {
	cases := []struct {
		in   string
		want any
	}{
		{"[]", []any{}},
		{"[1, 2, 3]", []any{1, 2, 3}},
		{"[a, b, c]", []any{"a", "b", "c"}},
		{"[1, 2, 3,]", []any{1, 2, 3}}, // trailing comma
		{"[ true , false ]", []any{true, false}},
		{`["x, y", z]`, []any{"x, y", "z"}},
		{"[[1, 2], [3]]", []any{[]any{1, 2}, []any{3}}},
	}
	for _, c := range cases {
		got, err := parseFlow(c.in, 1)
		require.NoError(t, err, "input %q", c.in)
		assert.Equal(t, c.want, got, "input %q", c.in)
	}
}

func TestParseFlowMapping(t *testing.T) {
	cases := []struct {
		in   string
		want any
	}{
		{"{}", map[string]any{}},
		{"{a: 1, b: 2}", map[string]any{"a": 1, "b": 2}},
		{"{ k : v }", map[string]any{"k": "v"}},
		{"{nested: {x: 1}}", map[string]any{"nested": map[string]any{"x": 1}}},
		{"{list: [1, 2]}", map[string]any{"list": []any{1, 2}}},
		{`{"q key": 1}`, map[string]any{"q key": 1}},
		{"{flag: }", map[string]any{"flag": nil}},
	}
	for _, c := range cases {
		got, err := parseFlow(c.in, 1)
		require.NoError(t, err, "input %q", c.in)
		assert.Equal(t, c.want, got, "input %q", c.in)
	}
}

func TestParseFlowErrors(t *testing.T) {
	for _, in := range []string{"[1, 2", "{a: 1", "[1, 2}", "{a: 1]", "[1] junk", "{a: 1, a: 2}"} {
		_, err := parseFlow(in, 1)
		assert.Error(t, err, "input %q", in)
	}
}

func TestFlowTrailingComment(t *testing.T) {
	got, err := parseFlow("[1, 2] # tail", 1)
	require.NoError(t, err)
	assert.Equal(t, []any{1, 2}, got)
}
