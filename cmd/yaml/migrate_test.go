package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateReindentsSpaces(t *testing.T) {
	in := "b: 2\na:\n  z: 1\n  y: 2\nlist:\n  - one\n  - two\n"
	out, err := run(t, in, "migrate")
	require.NoError(t, err)
	// Key order is preserved from the source (b before a, z before y), unlike fmt.
	assert.Equal(t, "b: 2\na:\n\tz: 1\n\ty: 2\nlist:\n\t- one\n\t- two\n", out)
}

func TestMigrateAcceptsFourSpaceIndent(t *testing.T) {
	in := "a:\n    b: 1\n    c:\n        - x\n        - y\n"
	out, err := run(t, in, "migrate")
	require.NoError(t, err)
	assert.Equal(t, "a:\n\tb: 1\n\tc:\n\t\t- x\n\t\t- y\n", out)
}

func TestMigratePreservesBlockScalarContent(t *testing.T) {
	in := "cmd: |\n  echo one\n  echo two\n"
	out, err := run(t, in, "migrate")
	require.NoError(t, err)
	assert.Contains(t, out, "echo one")
	assert.Contains(t, out, "echo two")
}

func TestMigrateRejectsAnchors(t *testing.T) {
	_, err := run(t, "a: &anchor 1\nb: *anchor\n", "migrate")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestMigrateRejectsMergeKeys(t *testing.T) {
	_, err := run(t, "base: &base\n  x: 1\nfoo:\n  <<: *base\n  y: 2\n", "migrate")
	require.Error(t, err)
}

func TestMigrateRejectsDuplicateKeys(t *testing.T) {
	_, err := run(t, "a: 1\na: 2\n", "migrate")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestMigrateWriteInPlace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.yaml")
	require.NoError(t, os.WriteFile(path, []byte("a:\n  b: 1\n"), 0o644))

	_, err := run(t, "", "migrate", "-w", path)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "a:\n\tb: 1\n", string(got))
}

func TestMigrateWriteRequiresFile(t *testing.T) {
	_, err := run(t, "a: 1\n", "migrate", "-w")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a file argument")
}
