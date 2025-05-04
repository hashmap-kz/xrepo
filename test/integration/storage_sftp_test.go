//go:build integration

package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hashmap-kz/xrepo/pkg/storage"

	"github.com/pkg/sftp"
	"github.com/stretchr/testify/assert"
)

const (
	root = "/home/testuser/tests"
)

func connectSFTP(t *testing.T) *sftp.Client {
	t.Helper()
	client, _ := createSftpClient("")
	return client
}

func prepareSFTPData(t *testing.T, client *sftp.Client, root string) {
	t.Helper()

	// Clean up and recreate test structure
	_ = client.RemoveDirectory(root)
	_ = client.Mkdir(root)

	// Create test dirs and files
	dirs := []string{"dir1", "dir2", "dir3/subdir"}
	for _, d := range dirs {
		err := client.MkdirAll(filepath.Join(root, d))
		assert.NoError(t, err)
	}

	files := []string{
		"dir1/file1.txt",
		"dir2/file2.txt",
		"dir3/subdir/file3.txt",
	}
	for _, f := range files {
		path := filepath.Join(root, f)
		file, err := client.Create(path)
		assert.NoError(t, err)
		_, _ = file.Write([]byte("data"))
		_ = file.Close()
	}
}

func TestSFTP_ListAll(t *testing.T) {
	client := connectSFTP(t)
	defer client.Close()

	prepareSFTPData(t, client, root)

	s := storage.NewSFTPStorage(client, root)
	files, err := s.ListAll(context.Background(), "")
	assert.NoError(t, err)

	// Expect all file paths (absolute or full, depending on impl)
	assert.GreaterOrEqual(t, len(files), 3)

	expected := []string{
		filepath.Join("dir1/file1.txt"),
		filepath.Join("dir2/file2.txt"),
		filepath.Join("dir3/subdir/file3.txt"),
	}
	for _, f := range expected {
		assert.Contains(t, files, f)
	}
}

func TestSFTP_ListTopLevelDirs(t *testing.T) {
	client := connectSFTP(t)
	defer client.Close()

	prepareSFTPData(t, client, root)

	s := storage.NewSFTPStorage(client, root)
	dirs, err := s.ListTopLevelDirs(context.Background(), root)
	assert.NoError(t, err)

	assert.True(t, dirs["dir1"])
	assert.True(t, dirs["dir2"])
	assert.True(t, dirs["dir3"])
	assert.Len(t, dirs, 3)
}
