package config

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

var (
	once   sync.Once
	config *Config
)

type (
	RepoCompressor string
	RepoType       string
	RepoEncryptor  string
)

const (
	RepoTypeLocal          RepoType       = "local"
	RepoTypeSFTP           RepoType       = "sftp"
	RepoTypeS3             RepoType       = "s3"
	RepoEncryptorAes256Gcm RepoEncryptor  = "aes-256-gcm"
	RepoCompressorGzip     RepoCompressor = "gzip"
	RepoCompressorZstd     RepoCompressor = "zstd"
)

type Config struct {
	// Repo main config
	RepoPath string   `json:"REPO_PATH"` // /mnt/backups
	RepoType RepoType `json:"REPO_TYPE"` // "local", "sftp", "s3"

	// Compression
	RepoCompressor RepoCompressor `json:"REPO_COMPRESSOR"` // gzip, zstd

	// Encryption
	RepoEncryptor      RepoEncryptor `json:"REPO_ENCRYPTOR"` // aes-256-gcm
	RepoEncryptionPass string        `json:"REPO_ENCRYPTION_PASS"`

	// SFTP Storage config
	RepoStorageSFTPHost                 string `json:"REPO_STORAGE_SFTP_HOST"`
	RepoStorageSFTPPort                 int    `json:"REPO_STORAGE_SFTP_PORT"`
	RepoStorageSFTPUser                 string `json:"REPO_STORAGE_SFTP_USER"`
	RepoStorageSFTPPass                 string `json:"REPO_STORAGE_SFTP_PASS"`
	RepoStorageSFTPPrivateKeyPath       string `json:"REPO_STORAGE_SFTP_PRIVATE_KEY_PATH"`
	RepoStorageSFTPPrivateKeyPassphrase string `json:"REPO_STORAGE_SFTP_PRIVATE_KEY_PASSPHRASE"`

	// S3 Storage config
	RepoStorageS3URL             string `json:"REPO_STORAGE_S3_URL"`
	RepoStorageS3AccessKeyID     string `json:"REPO_STORAGE_S3_ACCESS_KEY_ID"`
	RepoStorageS3SecretAccessKey string `json:"REPO_STORAGE_S3_SECRET_ACCESS_KEY"`
	RepoStorageS3Bucket          string `json:"REPO_STORAGE_S3_BUCKET"`
	RepoStorageS3Region          string `json:"REPO_STORAGE_S3_REGION"`
	RepoStorageS3UsePathStyle    bool   `json:"REPO_STORAGE_S3_USE_PATH_STYLE"`
	RepoStorageS3DisableSSL      bool   `json:"REPO_STORAGE_S3_DISABLE_SSL"`
}

// LoadConfigFromFile unmarshal file into config struct
func LoadConfigFromFile(filename string) *Config {
	once.Do(func() {
		loadFromFile(filename)
	})
	return config
}

// LoadConfig unmarshal raw data into config struct
func LoadConfig(content []byte) *Config {
	once.Do(func() {
		loadFromBuf(content)
	})
	return config
}

// helper internal functions, suitable for testing

func loadFromFile(filename string) *Config {
	content, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	content = expandEnvVars(content)

	var cfg Config
	err = json.Unmarshal(content, &cfg)
	if err != nil {
		log.Fatal(err)
	}
	config = &cfg
	return config
}

func loadFromBuf(content []byte) *Config {
	content = expandEnvVars(content)

	var cfg Config
	err := json.Unmarshal(content, &cfg)
	if err != nil {
		log.Fatal(err)
	}
	config = &cfg
	return config
}

func expandEnvVars(buf []byte) []byte {
	s := string(buf)
	e := os.ExpandEnv(s)
	return []byte(e)
}

func Cfg() *Config {
	if config == nil {
		log.Fatal("config was not loaded in main")
	}
	return config
}
