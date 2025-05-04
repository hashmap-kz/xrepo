package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var content = []byte(`
{
  "REPO_PATH": "./backups",
  "REPO_TYPE": "local",
  "REPO_COMPRESSOR": "gzip",
  "REPO_ENCRYPTOR": "aes-256-gcm",
  "REPO_ENCRYPTION_PASS": "102030pass5",
  "REPO_STORAGE_SFTP_HOST": "db-server.example.com",
  "REPO_STORAGE_SFTP_PORT": 22,
  "REPO_STORAGE_SFTP_USER": "backupuser",
  "REPO_STORAGE_SFTP_PASS": "",
  "REPO_STORAGE_SFTP_PRIVATE_KEY_PATH": "/home/operator/.ssh/id_rsa",
  "REPO_STORAGE_SFTP_PRIVATE_KEY_PASSPHRASE": "",
  "REPO_STORAGE_S3_URL": "http://10.40.240.189:9000",
  "REPO_STORAGE_S3_ACCESS_KEY_ID": "minioadmin",
  "REPO_STORAGE_S3_SECRET_ACCESS_KEY": "minioadmin123",
  "REPO_STORAGE_S3_BUCKET": "backups",
  "REPO_STORAGE_S3_REGION": "main",
  "REPO_STORAGE_S3_USE_PATH_STYLE": true,
  "REPO_STORAGE_S3_DISABLE_SSL": true,
  "LOG_LEVEL": "info",
  "LOG_DIR": "/var/log/pgbackup"
}
`)

func TestConfigStructureFromBytes(t *testing.T) {
	require.NoError(t, os.Setenv("PKEY_PASS", "my-super-secret-pass-1"))
	require.NoError(t, os.Setenv("BACKUPUSER_PASS", "102030QW.1"))

	cfg := loadFromBuf(content)

	assert.Equal(t, "./backups", cfg.RepoPath)
	assert.Equal(t, RepoTypeLocal, cfg.RepoType)
	assert.Equal(t, "gzip", string(cfg.RepoCompressor))
	assert.Equal(t, "aes-256-gcm", string(cfg.RepoEncryptor))
	assert.Equal(t, "102030pass5", cfg.RepoEncryptionPass)

	assert.Equal(t, "db-server.example.com", cfg.RepoStorageSFTPHost)
	assert.Equal(t, 22, cfg.RepoStorageSFTPPort)
	assert.Equal(t, "backupuser", cfg.RepoStorageSFTPUser)
	assert.Equal(t, "", cfg.RepoStorageSFTPPass)
	assert.Equal(t, "/home/operator/.ssh/id_rsa", cfg.RepoStorageSFTPPrivateKeyPath)
	assert.Equal(t, "", cfg.RepoStorageSFTPPrivateKeyPassphrase)

	assert.Equal(t, "http://10.40.240.189:9000", cfg.RepoStorageS3URL)
	assert.Equal(t, "minioadmin", cfg.RepoStorageS3AccessKeyID)
	assert.Equal(t, "minioadmin123", cfg.RepoStorageS3SecretAccessKey)
	assert.Equal(t, "backups", cfg.RepoStorageS3Bucket)
	assert.Equal(t, "main", cfg.RepoStorageS3Region)
	assert.True(t, cfg.RepoStorageS3UsePathStyle)
	assert.True(t, cfg.RepoStorageS3DisableSSL)
}

func TestConfigStructureFromFile(t *testing.T) {
	require.NoError(t, os.Setenv("PKEY_PASS", "my-super-secret-pass-2"))
	require.NoError(t, os.Setenv("BACKUPUSER_PASS", "102030QW.2"))

	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "config.json")
	require.NoError(t, os.WriteFile(filePath, content, 0o600))

	cfg := loadFromFile(filePath)

	assert.Equal(t, "./backups", cfg.RepoPath)
	assert.Equal(t, RepoTypeLocal, cfg.RepoType)
	assert.Equal(t, "gzip", string(cfg.RepoCompressor))
	assert.Equal(t, "aes-256-gcm", string(cfg.RepoEncryptor))
	assert.Equal(t, "102030pass5", cfg.RepoEncryptionPass)

	assert.Equal(t, "db-server.example.com", cfg.RepoStorageSFTPHost)
	assert.Equal(t, 22, cfg.RepoStorageSFTPPort)
	assert.Equal(t, "backupuser", cfg.RepoStorageSFTPUser)
	assert.Equal(t, "", cfg.RepoStorageSFTPPass)
	assert.Equal(t, "/home/operator/.ssh/id_rsa", cfg.RepoStorageSFTPPrivateKeyPath)
	assert.Equal(t, "", cfg.RepoStorageSFTPPrivateKeyPassphrase)

	assert.Equal(t, "http://10.40.240.189:9000", cfg.RepoStorageS3URL)
	assert.Equal(t, "minioadmin", cfg.RepoStorageS3AccessKeyID)
	assert.Equal(t, "minioadmin123", cfg.RepoStorageS3SecretAccessKey)
	assert.Equal(t, "backups", cfg.RepoStorageS3Bucket)
	assert.Equal(t, "main", cfg.RepoStorageS3Region)
	assert.True(t, cfg.RepoStorageS3UsePathStyle)
	assert.True(t, cfg.RepoStorageS3DisableSSL)
}
