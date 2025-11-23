package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")

	// Write initial content
	err := AtomicWriteString(path, "initial", 0o600)
	require.NoError(t, err)

	// Verify content
	data, err := os.ReadFile(path) //nolint:gosec // G304: Test file path
	require.NoError(t, err)
	assert.Equal(t, "initial", string(data))

	// Overwrite
	err = AtomicWriteString(path, "updated", 0o600)
	require.NoError(t, err)

	data, err = os.ReadFile(path) //nolint:gosec // G304: Test file path
	require.NoError(t, err)
	assert.Equal(t, "updated", string(data))

	// Verify no temp files left
	entries, _ := os.ReadDir(tmpDir)
	assert.Len(t, entries, 1) // Only test.txt, no .tmp-*
}

func TestAtomicWritePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")

	err := AtomicWriteString(path, "content", 0o600)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestAtomicWriteBinary(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.bin")

	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	err := AtomicWrite(path, binaryData, 0o600)
	require.NoError(t, err)

	data, err := os.ReadFile(path) //nolint:gosec // G304: Test file path
	require.NoError(t, err)
	assert.Equal(t, binaryData, data)
}
