//go:build integration

package integration

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/hashmap-kz/xrepo/pkg/repo"
	"github.com/hashmap-kz/xrepo/pkg/storage"

	"github.com/hashmap-kz/streamcrypt/pkg/crypt/aesgcm"

	"github.com/pkg/sftp"
	"github.com/stretchr/testify/assert"

	"github.com/hashmap-kz/streamcrypt/pkg/codec"

	"github.com/stretchr/testify/require"
)

// --- Helpers ---

func repoTestsCreateSftpClient(t *testing.T) (*sftp.Client, storage.Storage) {
	t.Helper()
	return createSftpClient("repo-unittest")
}

func TestRepoSFTP_PutAndGet_GzipV1(t *testing.T) {
	_, stor := repoTestsCreateSftpClient(t)

	gzipCompressor := &codec.GzipCompressor{}

	r := repo.NewWriteReader(stor, gzipCompressor, nil)

	content := []byte("This is a test backup payload.")
	logicalPath := "pg_data/PG_VERSION"

	// Act - store file
	_, err := r.PutObject(context.Background(), logicalPath, bytes.NewReader(content))
	require.NoError(t, err)

	// Act - retrieve and decompress file
	reader, err := r.ReadObject(context.Background(), logicalPath)
	require.NoError(t, err)

	result, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.NoError(t, reader.Close())
	assert.Equal(t, content, result)
}

func TestRepoSFTP_PutAndGet_NoCompression_NoEncryption(t *testing.T) {
	_, stor := repoTestsCreateSftpClient(t)
	r := repo.NewWriteReader(stor, nil, nil)

	content := []byte("raw test data")
	logicalPath := "plain/data.txt"

	// Write
	_, err := r.PutObject(context.Background(), logicalPath, bytes.NewReader(content))
	require.NoError(t, err)

	// Read
	rc, err := r.ReadObject(context.Background(), logicalPath)
	require.NoError(t, err)

	result := readAllAndClose(t, rc)
	assert.Equal(t, content, result)
}

func TestRepoSFTP_PutAndGet_GzipV2(t *testing.T) {
	_, stor := repoTestsCreateSftpClient(t)
	r := repo.NewWriteReader(stor, &codec.GzipCompressor{}, nil)

	content := []byte("gzip compressed data test")
	logicalPath := "gzip/data.txt"

	finalPath, err := r.PutObject(context.Background(), logicalPath, bytes.NewReader(content))
	require.NoError(t, err)

	rc, err := r.ReadObject(context.Background(), logicalPath)
	require.NoError(t, err)

	result := readAllAndClose(t, rc)
	assert.Equal(t, content, result)

	assert.True(t, strings.HasSuffix(finalPath, ".gz"))
}

func TestRepoSFTP_PutAndGet_EncryptionOnly(t *testing.T) {
	_, stor := repoTestsCreateSftpClient(t)
	crypter := aesgcm.NewChunkedGCMCrypter("secret-key")
	r := repo.NewWriteReader(stor, nil, crypter)

	content := []byte("encrypted only data test")
	logicalPath := "enc/data.txt"

	finalPath, err := r.PutObject(context.Background(), logicalPath, bytes.NewReader(content))
	require.NoError(t, err)

	rc, err := r.ReadObject(context.Background(), logicalPath)
	require.NoError(t, err)

	result := readAllAndClose(t, rc)
	assert.Equal(t, content, result)

	assert.True(t, strings.HasSuffix(finalPath, ".aes"))
}

func TestRepoSFTP_PutAndGet_GzipAndEncryption(t *testing.T) {
	_, stor := repoTestsCreateSftpClient(t)
	crypter := aesgcm.NewChunkedGCMCrypter("combo-key")
	compressor := &codec.GzipCompressor{}

	r := repo.NewWriteReader(stor, compressor, crypter)

	content := []byte("gzip + encrypted")
	logicalPath := "combo/file.txt"

	finalPath, err := r.PutObject(context.Background(), logicalPath, bytes.NewReader(content))
	require.NoError(t, err)

	rc, err := r.ReadObject(context.Background(), logicalPath)
	require.NoError(t, err)

	result := readAllAndClose(t, rc)
	assert.Equal(t, content, result)

	assert.True(t, strings.HasSuffix(finalPath, ".gz.aes"))
}

func TestRepoSFTP_EmptyFile(t *testing.T) {
	_, stor := repoTestsCreateSftpClient(t)
	r := repo.NewWriteReader(stor, &codec.GzipCompressor{}, nil)

	var empty bytes.Buffer

	finalPath, err := r.PutObject(context.Background(), "empty/file.txt", &empty)
	require.NoError(t, err)

	rc, err := r.ReadObject(context.Background(), "empty/file.txt")
	require.NoError(t, err)

	result := readAllAndClose(t, rc)
	assert.Empty(t, result)

	assert.True(t, strings.HasSuffix(finalPath, ".gz"))
}
