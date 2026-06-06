package yaml

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePlain(t *testing.T) {
	cases := []struct {
		in   string
		want any
	}{
		{"", nil},
		{"~", nil},
		{"null", nil},
		{"NULL", nil},
		{"true", true},
		{"TRUE", true},
		{"false", false},
		{"0", 0},
		{"123", 123},
		{"+5", 5},
		{"-9", -9},
		{"0x10", 16},
		{"0o10", 8},
		{"1.5", 1.5},
		{"-2.0", -2.0},
		{"1e3", 1000.0},
		{"hello", "hello"},
		{"yes", "yes"}, // not a bool in the 1.2 core schema
		{"007", "007"}, // leading zero is not a valid int -> string
		{"12:30", "12:30"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, resolvePlain(c.in), "input %q", c.in)
	}
}

func TestResolvePlainInfNan(t *testing.T) {
	assert.Equal(t, math.Inf(1), resolvePlain(".inf"))
	assert.Equal(t, math.Inf(-1), resolvePlain("-.inf"))
	f, ok := resolvePlain(".nan").(float64)
	require.True(t, ok)
	assert.True(t, math.IsNaN(f))
}

func TestStripComment(t *testing.T) {
	cases := []struct{ in, want string }{
		{"value # comment", "value"},
		{"value#nospace", "value#nospace"},
		{"# whole", ""},
		{"a\t# tabbed", "a"},
		{"no comment", "no comment"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, stripComment(c.in), "input %q", c.in)
	}
}

func TestDoubleQuoteEscapes(t *testing.T) {
	cases := []struct{ in, want string }{
		{`"a\nb"`, "a\nb"},
		{`"tab\there"`, "tab\there"},
		{`"quote\"inside"`, `quote"inside`},
		{`"back\\slash"`, `back\slash`},
		{`"A"`, "A"},
		{`"\x41"`, "A"},
		{`"null\0byte"`, "null\x00byte"},
	}
	for _, c := range cases {
		got, err := resolveScalar(c.in, 1)
		require.NoError(t, err, "input %q", c.in)
		assert.Equal(t, c.want, got, "input %q", c.in)
	}
}

func TestSingleQuoteEscapes(t *testing.T) {
	got, err := resolveScalar(`'it''s fine'`, 1)
	require.NoError(t, err)
	assert.Equal(t, "it's fine", got)
}

func TestQuotedScalarErrors(t *testing.T) {
	for _, in := range []string{`"unterminated`, `'unterminated`, `"bad\q"`, `"trunc\u00"`} {
		_, err := resolveScalar(in, 1)
		assert.Error(t, err, "input %q", in)
	}
}

func TestQuotedThenComment(t *testing.T) {
	got, err := resolveScalar(`"x" # note`, 1)
	require.NoError(t, err)
	assert.Equal(t, "x", got)
}

func TestIsBlockScalarHeader(t *testing.T) {
	yes := []string{"|", ">", "|-", "|+", ">-", ">+", "| # comment", ">  "}
	no := []string{"", "x", "|x", "abc", "[1]"}
	for _, h := range yes {
		assert.True(t, isBlockScalarHeader(h), "header %q", h)
	}
	for _, h := range no {
		assert.False(t, isBlockScalarHeader(h), "header %q", h)
	}
}
