package boot

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashmap-kz/xrepo/config"
	"github.com/stretchr/testify/assert"
)

func TestBoot_LocalRepo(t *testing.T) {
	tempDir := t.TempDir()
	prefix := "local-repo"
	repo, err := DecideRepo(&config.Config{
		RepoPath: tempDir,
		RepoType: config.RepoTypeLocal,
	}, prefix)
	assert.NoError(t, err)

	listAll, err := repo.ListAll(context.TODO(), "")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(listAll))

	fileContent := []byte("content")
	object, err := repo.PutObject(context.TODO(), "my-file.txt", bytes.NewReader(fileContent))
	assert.NoError(t, err)
	assert.True(t, strings.HasSuffix(filepath.ToSlash(object), "my-file.txt"))

	readCloser, err := repo.ReadObject(context.TODO(), "my-file.txt")
	assert.NoError(t, err)
	defer readCloser.Close()

	readAll, err := io.ReadAll(readCloser)
	assert.NoError(t, err)
	assert.Equal(t, readAll, fileContent)

	all, err := repo.ListAll(context.TODO(), "")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(all))
}

func TestBoot_LocalRepoWriteWithPrefix(t *testing.T) {
	tempDir := t.TempDir()
	prefix := "local-repo"
	repo, err := DecideRepo(&config.Config{
		RepoPath: tempDir,
		RepoType: config.RepoTypeLocal,
	}, prefix)
	assert.NoError(t, err)

	listAll, err := repo.ListAll(context.TODO(), "")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(listAll))

	fileContent := []byte("content")
	object, err := repo.PutObject(context.TODO(), "a/b/c/my-file.txt", bytes.NewReader(fileContent))
	assert.NoError(t, err)
	assert.True(t, strings.HasSuffix(filepath.ToSlash(object), "a/b/c/my-file.txt"))

	readCloser, err := repo.ReadObject(context.TODO(), "a/b/c/my-file.txt")
	assert.NoError(t, err)
	defer readCloser.Close()

	readAll, err := io.ReadAll(readCloser)
	assert.NoError(t, err)
	assert.Equal(t, readAll, fileContent)

	all, err := repo.ListAll(context.TODO(), "a/b/c/")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(all))
}
