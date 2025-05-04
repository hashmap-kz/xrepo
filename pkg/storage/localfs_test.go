package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashmap-kz/xrepo/pkg/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStorage_PutAndReadObject(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocal(dir)
	assert.NoError(t, err)

	content := []byte("hello, pgwal!")
	path := "test/pg/put.txt"

	err = s.PutObject(context.Background(), path, bytes.NewReader(content))
	assert.NoError(t, err)

	rc, err := s.ReadObject(context.Background(), path)
	assert.NoError(t, err)
	defer rc.Close()

	readContent, err := io.ReadAll(rc)
	assert.NoError(t, err)
	assert.Equal(t, content, readContent)
}

func TestLocalStorage_Exists(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocal(dir)
	assert.NoError(t, err)

	path := "exists.txt"
	exists, err := s.Exists(context.Background(), path)
	assert.NoError(t, err)
	assert.False(t, exists)

	err = s.PutObject(context.Background(), path, bytes.NewBufferString("check me"))
	assert.NoError(t, err)

	exists, err = s.Exists(context.Background(), path)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestLocalStorage_SHA256(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocal(dir)
	assert.NoError(t, err)

	content := []byte("hash me")
	path := "hash/val.txt"

	err = s.PutObject(context.Background(), path, bytes.NewReader(content))
	assert.NoError(t, err)

	hash, err := s.SHA256(context.Background(), path)
	assert.NoError(t, err)

	expectedHash := common.Sha256FromBytes(content)
	assert.Equal(t, expectedHash, hash)
}

func TestLocalStorage_ListAll_1(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocal(dir)
	assert.NoError(t, err)

	err = s.PutObject(context.Background(), "a/file1.txt", bytes.NewReader([]byte("1")))
	assert.NoError(t, err)

	err = s.PutObject(context.Background(), "a/file2.txt", bytes.NewReader([]byte("2")))
	assert.NoError(t, err)

	err = s.PutObject(context.Background(), "b/file3.txt", bytes.NewReader([]byte("3")))
	assert.NoError(t, err)

	files, err := s.ListAll(context.Background(), "")
	assert.NoError(t, err)

	assert.ElementsMatch(t, []string{
		"a/file1.txt",
		"a/file2.txt",
		"b/file3.txt",
	}, files)
}

func TestLocalStorage_ListAll_2(t *testing.T) {
	tmpDir := t.TempDir()
	storage := &localStorage{baseDir: tmpDir}

	// Create nested test files
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "20250414/pg_data/base"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "20250414/pg_data/base/1234"), []byte("hello"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "20250414/pg_data/base/5678"), []byte("world"), 0o600))

	paths, err := storage.ListAll(context.Background(), "20250414")
	assert.NoError(t, err)
	assert.Len(t, paths, 2)
	assert.Contains(t, paths[0], "1234")
	assert.Contains(t, paths[1], "5678")
}

func TestLocalStorage_ListTopLevelDirs(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocal(dir)
	assert.NoError(t, err)

	err = os.MkdirAll(filepath.Join(dir, "x/sub"), common.DirPerm)
	assert.NoError(t, err)

	err = os.MkdirAll(filepath.Join(dir, "y"), common.DirPerm)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("ignore"), 0o600)
	assert.NoError(t, err)

	dirs, err := s.ListTopLevelDirs(context.Background(), dir)
	assert.NoError(t, err)

	assert.Equal(t, map[string]bool{
		"x": true,
		"y": true,
	}, dirs)
}
