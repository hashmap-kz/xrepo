//go:build integration

package integration

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/hashmap-kz/xrepo/pkg/storage"

	"github.com/hashmap-kz/xrepo/pkg/s3x"
	"github.com/hashmap-kz/xrepo/pkg/sftpx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSftp_Connect(t *testing.T) {
	pkeyPath := "./environ/files/dotfiles/.ssh/id_ed25519"

	err := os.Chmod(pkeyPath, 0o600)
	require.NoError(t, err)

	client, err := sftpx.NewSFTPClient(&sftpx.SFTPConfig{
		Host:     "localhost",
		Port:     "2323",
		User:     "testuser",
		PkeyPath: pkeyPath,
	})
	assert.NoError(t, err)
	defer client.Close()
}

func TestS3_Connect(t *testing.T) {
	client, err := s3x.NewS3Storage(&s3x.S3Config{
		EndpointURL:     "https://localhost:9000",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin123",
		Bucket:          "backups",
		Region:          "main",
		UsePathStyle:    true,
		DisableSSL:      true,
	})
	require.NoError(t, err)

	stor := storage.NewS3Storage(client.Client(), "backups", "demo")
	data := bytes.NewReader([]byte("content"))

	err = stor.PutObject(context.TODO(), "temp1", data)
	require.NoError(t, err)

	r, err := stor.ReadObject(context.TODO(), "temp1")
	require.NoError(t, err)
	defer r.Close()

	dataResult, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, dataResult, []byte("content"))
}
