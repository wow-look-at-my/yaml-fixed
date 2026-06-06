package tabyaml

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalScalarsAndMaps(t *testing.T) {
	cases := []struct {
		in	any
		want	string
	}{
		{nil, "null\n"},
		{42, "42\n"},
		{true, "true\n"},
		{"plain", "plain\n"},
		{3.5, "3.5\n"},
		{map[string]any{"a": 1, "b": "x"}, "a: 1\nb: x\n"},
		{[]any{1, 2, 3}, "- 1\n- 2\n- 3\n"},
		{map[string]any{}, "{}\n"},
		{[]any{}, "[]\n"},
	}
	for _, c := range cases {
		got, err := Marshal(c.in)
		require.Nil(t, err)

		assert.Equal(t, c.want, string(got))

	}
}

func TestMarshalNestedUsesTabs(t *testing.T) {
	in := map[string]any{
		"server": map[string]any{"host": "localhost", "port": 8080},
	}
	got, err := Marshal(in)
	require.Nil(t, err)

	want := "server:\n\thost: localhost\n\tport: 8080\n"
	assert.Equal(t, want, string(got))

}

func TestMarshalSequenceOfMappingsExpanded(t *testing.T) {
	in := []any{map[string]any{"name": "Alice"}}
	got, err := Marshal(in)
	require.Nil(t, err)

	want := "-\n\tname: Alice\n"
	assert.Equal(t, want, string(got))

}

func TestMarshalQuotesAmbiguousStrings(t *testing.T) {
	cases := []struct {
		in	string
		want	string
	}{
		{"true", `x: "true"` + "\n"},
		{"123", `x: "123"` + "\n"},
		{"", `x: ""` + "\n"},
		{"has: colon", `x: "has: colon"` + "\n"},
		{"with\nnewline", `x: "with\nnewline"` + "\n"},
		{"safe", "x: safe\n"},
	}
	for _, c := range cases {
		got, err := Marshal(map[string]any{"x": c.in})
		require.Nil(t, err)

		assert.Equal(t, c.want, string(got))

	}
}

func TestMarshalStructFieldOrderAndTags(t *testing.T) {
	type Server struct {
		Host	string		`yaml:"host"`
		Port	int		`yaml:"port"`
		Tags	[]string	`yaml:"tags,omitempty"`
		secret	string		// unexported: skipped
		Skip	string		`yaml:"-"`
		Comment	string		`yaml:"comment,omitempty"`
	}
	s := Server{Host: "h", Port: 80, secret: "x", Skip: "y"}
	_ = s.secret
	got, err := Marshal(s)
	require.Nil(t, err)

	want := "host: h\nport: 80\n"
	assert.Equal(t, want, string(got))

}

func TestMarshalFloatSpecials(t *testing.T) {
	got, err := Marshal(map[string]any{"a": 1.0})
	require.Nil(t, err)

	assert.Equal(t, "a: 1.0\n", string(got))

}

func TestMarshalPointer(t *testing.T) {
	n := 5
	got, err := Marshal(map[string]any{"p": &n})
	require.Nil(t, err)

	assert.Equal(t, "p: 5\n", string(got))

}
