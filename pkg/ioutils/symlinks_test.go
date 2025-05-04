//go:build !windows

package ioutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- LOCAL FS ---

func TestResolveSymlinkIfNeededLocalfs_RealFile(t *testing.T) {
	dir := t.TempDir()

	realFile := filepath.Join(dir, "data.txt")
	require.NoError(t, os.WriteFile(realFile, []byte("hello"), 0o600))

	resolved, err := ResolveSymlinkIfNeededLocalfs(realFile)
	require.NoError(t, err)
	assert.Equal(t, realFile, resolved)
}

func TestResolveSymlinkIfNeededLocalfs_Symlink(t *testing.T) {
	dir := t.TempDir()

	realFile := filepath.Join(dir, "data.txt")
	symFile := filepath.Join(dir, "link.txt")

	require.NoError(t, os.WriteFile(realFile, []byte("hello"), 0o600))
	require.NoError(t, os.Symlink(realFile, symFile))

	resolved, err := ResolveSymlinkIfNeededLocalfs(symFile)
	require.NoError(t, err)
	assert.Equal(t, realFile, resolved)
}

func TestResolveSymlinkIfNeededLocalfs_BrokenSymlink(t *testing.T) {
	dir := t.TempDir()
	badLink := filepath.Join(dir, "broken")

	require.NoError(t, os.Symlink("/nonexistent", badLink))

	_, err := ResolveSymlinkIfNeededLocalfs(badLink)
	assert.Error(t, err)
}

func TestResolveSymlinkIfNeededLocalfs_FileNotExist(t *testing.T) {
	_, err := ResolveSymlinkIfNeededLocalfs("/does/not/exist")
	assert.Error(t, err)
}
