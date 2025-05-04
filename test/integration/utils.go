package integration

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	storage2 "github.com/hashmap-kz/xrepo/pkg/storage"

	"github.com/hashmap-kz/xrepo/pkg/s3x"
	"github.com/hashmap-kz/xrepo/pkg/sftpx"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/sftp"
	"github.com/stretchr/testify/require"
)

const (
	pkeyPath      = "environ/files/dotfiles/.ssh/id_ed25519"
	tmpWorkingDir = "/tmp/integ-da4e0a1a-8adb-4e2d-b805-2ab72bfa25d0"
	connStr       = "postgres://postgres:postgres@localhost:15433/postgres"
)

var pgData = filepath.ToSlash(filepath.Join(tmpWorkingDir, "pgdata"))

func createS3Client(prefix string) (*s3.Client, storage2.Storage) {
	client, err := s3x.NewS3Storage(&s3x.S3Config{
		EndpointURL:     "https://localhost:9000",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin123",
		Bucket:          "backups",
		Region:          "main",
		UsePathStyle:    true,
		DisableSSL:      true,
	})
	if err != nil {
		log.Fatal(err)
	}

	store := storage2.NewS3Storage(client.Client(), "backups", prefix)
	return client.Client(), store
}

func createSftpClient(prefix string) (*sftp.Client, storage2.Storage) {
	pkeyPath := "./environ/files/dotfiles/.ssh/id_ed25519"
	err := os.Chmod(pkeyPath, 0o600)
	if err != nil {
		log.Fatal(err)
	}

	client, err := sftpx.NewSFTPClient(&sftpx.SFTPConfig{
		Host:     "localhost",
		Port:     "2323",
		User:     "testuser",
		PkeyPath: pkeyPath,
	})
	if err != nil {
		log.Fatal(err)
	}

	store := storage2.NewSFTPStorage(client.SFTPClient(), prefix)
	return client.SFTPClient(), store
}

func readAllAndClose(t *testing.T, r io.ReadCloser) []byte {
	t.Helper()
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return data
}
