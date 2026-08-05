package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalStruct(t *testing.T) {
	type TLS struct {
		Enabled bool `yaml:"enabled"`
	}
	type Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
		TLS  TLS    `yaml:"tls"`
		Tags []string
	}
	in := "host: localhost\nport: 8080\ntls:\n\tenabled: true\ntags:\n\t- a\n\t- b\n"
	var s Server
	require.NoError(t, Unmarshal([]byte(in), &s))
	want := Server{Host: "localhost", Port: 8080, TLS: TLS{Enabled: true}, Tags: []string{"a", "b"}}
	assert.Equal(t, want, s)
}

func TestUnmarshalIntoMap(t *testing.T) {
	var m map[string]int
	require.NoError(t, Unmarshal([]byte("a: 1\nb: 2\n"), &m))
	assert.Equal(t, map[string]int{"a": 1, "b": 2}, m)
}

func TestUnmarshalIntoSlice(t *testing.T) {
	var s []int
	require.NoError(t, Unmarshal([]byte("- 1\n- 2\n- 3\n"), &s))
	assert.Equal(t, []int{1, 2, 3}, s)
}

func TestUnmarshalIntoInterface(t *testing.T) {
	var v any
	require.NoError(t, Unmarshal([]byte("a: 1\n"), &v))
	assert.Equal(t, map[string]any{"a": 1}, v)
}

func TestUnmarshalScalarTypes(t *testing.T) {
	var i int
	require.NoError(t, Unmarshal([]byte("42"), &i))
	assert.Equal(t, 42, i)

	var f float64
	require.NoError(t, Unmarshal([]byte("3"), &f))
	assert.Equal(t, 3.0, f)

	var b bool
	require.NoError(t, Unmarshal([]byte("true"), &b))
	assert.True(t, b)

	var u uint8
	require.NoError(t, Unmarshal([]byte("200"), &u))
	assert.Equal(t, uint8(200), u)

	var s string
	require.NoError(t, Unmarshal([]byte("hi"), &s))
	assert.Equal(t, "hi", s)
}

func TestUnmarshalScalarCoercedIntoString(t *testing.T) {
	// An unquoted scalar decoding into a string field uses its canonical
	// text, matching gopkg.in/yaml.v3: a caller writing `cmd: true` (a shell
	// command that happens to be the word "true") expects the string "true",
	// not a type error.
	cases := []struct {
		in   string
		want string
	}{
		{"true", "true"},
		{"false", "false"},
		{"42", "42"},
		{"3.5", "3.5"},
	}
	for _, tc := range cases {
		var s string
		require.NoError(t, Unmarshal([]byte(tc.in), &s), "input %q", tc.in)
		assert.Equal(t, tc.want, s, "input %q", tc.in)
	}
}

func TestUnmarshalMappingIntoStringErrors(t *testing.T) {
	// A collection has no scalar text to coerce.
	var s string
	assert.Error(t, Unmarshal([]byte("a: 1\n"), &s))
	assert.Error(t, Unmarshal([]byte("- 1\n"), &s))
}

func TestUnmarshalPointerField(t *testing.T) {
	type T struct {
		P *int `yaml:"p"`
	}
	var v T
	require.NoError(t, Unmarshal([]byte("p: 7"), &v))
	require.NotNil(t, v.P)
	assert.Equal(t, 7, *v.P)
}

func TestUnmarshalErrors(t *testing.T) {
	var i int
	assert.Error(t, Unmarshal([]byte("notanumber"), &i))

	assert.Error(t, Unmarshal([]byte("1"), i)) // non-pointer target

	var u uint
	assert.Error(t, Unmarshal([]byte("-1"), &u)) // negative into uint

	var i8 int8
	assert.Error(t, Unmarshal([]byte("9999"), &i8)) // overflow
}

func TestUnmarshalUnknownKeysIgnored(t *testing.T) {
	type T struct {
		A int `yaml:"a"`
	}
	var v T
	require.NoError(t, Unmarshal([]byte("a: 1\nb: 2"), &v))
	assert.Equal(t, 1, v.A)
}

func TestUnmarshalNullClearsPointer(t *testing.T) {
	type T struct {
		P *int `yaml:"p"`
	}
	v := T{P: new(int)}
	require.NoError(t, Unmarshal([]byte("p: null"), &v))
	assert.Nil(t, v.P)
}
