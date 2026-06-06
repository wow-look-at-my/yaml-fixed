package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wow-look-at-my/yaml-fixed/yaml"
)

// run executes the root command with the given stdin and arguments, returning
// combined stdout/stderr output and the resulting error.
func run(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	fmtWrite = false // reset persistent flag state between runs
	var out strings.Builder
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetIn(strings.NewReader(stdin))
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return out.String(), err
}

func TestValidateOK(t *testing.T) {
	out, err := run(t, "a: 1\nb:\n\t- x\n", "validate")
	require.NoError(t, err)
	assert.Equal(t, "ok\n", out)
}

func TestValidateRejectsSpaces(t *testing.T) {
	_, err := run(t, "a:\n  b: 1\n", "validate")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spaces cannot be used for indentation")
}

func TestToJSON(t *testing.T) {
	out, err := run(t, "a: 1\nb:\n\t- x\n\t- y\n", "to-json")
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":1,"b":["x","y"]}`, out)
}

func TestFromJSON(t *testing.T) {
	out, err := run(t, `{"a":1,"b":["x","y"]}`, "from-json")
	require.NoError(t, err)
	assert.Equal(t, "a: 1\nb:\n\t- x\n\t- y\n", out)
}

func TestFromJSONKeepsIntegers(t *testing.T) {
	out, err := run(t, `{"port":8080}`, "from-json")
	require.NoError(t, err)
	assert.Equal(t, "port: 8080\n", out)
}

// from-json parses JSON with the YAML parser itself (one parser, no separate
// JSON path), so pretty-printed, space-indented, multi-line JSON converts too.
func TestFromJSONPrettyPrinted(t *testing.T) {
	prev := yaml.Warn
	yaml.Warn = func(string) {} // silence the once-per-file space warning
	t.Cleanup(func() { yaml.Warn = prev })

	in := "{\n  \"a\": 1,\n  \"b\": [\"x\", \"y\"]\n}\n"
	out, err := run(t, in, "from-json")
	require.NoError(t, err)
	assert.Equal(t, "a: 1\nb:\n\t- x\n\t- y\n", out)
}

func TestFmtCanonicalises(t *testing.T) {
	// Keys come out sorted and re-indented with tabs.
	out, err := run(t, "b: 2\na:\n\tz: 1\n", "fmt")
	require.NoError(t, err)
	assert.Equal(t, "a:\n\tz: 1\nb: 2\n", out)
}

func TestFmtWriteInPlace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.yaml")
	require.NoError(t, os.WriteFile(path, []byte("b: 2\na: 1\n"), 0o644))

	_, err := run(t, "", "fmt", "-w", path)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "a: 1\nb: 2\n", string(got))
}

func TestFmtWriteRequiresFile(t *testing.T) {
	_, err := run(t, "a: 1\n", "fmt", "-w")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a file argument")
}

func TestReadFromFileArgument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.yaml")
	require.NoError(t, os.WriteFile(path, []byte("k: v\n"), 0o644))

	out, err := run(t, "", "to-json", path)
	require.NoError(t, err)
	assert.JSONEq(t, `{"k":"v"}`, out)
}

func TestMissingFile(t *testing.T) {
	_, err := run(t, "", "validate", filepath.Join(t.TempDir(), "nope.yaml"))
	require.Error(t, err)
}

func TestInvalidJSON(t *testing.T) {
	_, err := run(t, "{not json", "from-json")
	require.Error(t, err)
}
