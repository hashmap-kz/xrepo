package boot

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/hashmap-kz/xrepo/config"
	"github.com/hashmap-kz/xrepo/pkg/clients/s3x"
	"github.com/hashmap-kz/xrepo/pkg/clients/sftpx"
	"github.com/hashmap-kz/xrepo/pkg/codec"
	"github.com/hashmap-kz/xrepo/pkg/crypt"
	"github.com/hashmap-kz/xrepo/pkg/crypt/aesgcm"
	"github.com/hashmap-kz/xrepo/pkg/repo"
	"github.com/hashmap-kz/xrepo/pkg/storage"
)

// DecideRepo inits repository with storage/compression/encryption assigned according to configs
func DecideRepo(dir string) (repo.WriteReader, error) {
	cfg := config.Cfg()
	baseDir := filepath.ToSlash(filepath.Join(cfg.RepoPath, dir))
	compressor, crypter := decideCompressorEncryptor(cfg)

	switch cfg.RepoType {
	// local
	case config.RepoTypeLocal:
		slog.Info("init local storage",
			slog.String("module", "boot"),
			slog.String("local storage ready with location", filepath.ToSlash(baseDir)),
		)
		local, err := storage.NewLocal(baseDir)
		if err != nil {
			return nil, err
		}
		return repo.NewWriteReader(local, compressor, crypter), nil

		// sftp
	case config.RepoTypeSFTP:
		slog.Info("init SFTP storage",
			slog.String("module", "boot"),
			slog.String("SFTP storage ready with location", filepath.ToSlash(baseDir)),
		)
		c, err := sftpx.NewSFTPClient(&sftpx.SFTPConfig{
			Host:       cfg.RepoStorageSFTPHost,
			Port:       fmt.Sprintf("%d", cfg.RepoStorageSFTPPort),
			User:       cfg.RepoStorageSFTPUser,
			PkeyPath:   cfg.RepoStorageSFTPPrivateKeyPath,
			Passphrase: cfg.RepoStorageSFTPPrivateKeyPassphrase,
		})
		if err != nil {
			return nil, err
		}
		sftpStorage := storage.NewSFTPStorage(c.SFTPClient(), baseDir)
		return repo.NewWriteReader(sftpStorage, compressor, crypter), nil

		// s3
	case config.RepoTypeS3:
		slog.Info("init s3 storage",
			slog.String("module", "boot"),
			slog.String("s3 storage ready with location", filepath.ToSlash(baseDir)),
		)
		c, err := s3x.NewS3Storage(&s3x.S3Config{
			EndpointURL:     cfg.RepoStorageS3URL,
			AccessKeyID:     cfg.RepoStorageS3AccessKeyID,
			SecretAccessKey: cfg.RepoStorageS3SecretAccessKey,
			Bucket:          cfg.RepoStorageS3Bucket,
			Region:          cfg.RepoStorageS3Region,
			UsePathStyle:    cfg.RepoStorageS3UsePathStyle,
			DisableSSL:      cfg.RepoStorageS3DisableSSL,
		})
		if err != nil {
			return nil, err
		}
		s3Storage := storage.NewS3Storage(c.Client(), cfg.RepoStorageS3Bucket, baseDir)
		return repo.NewWriteReader(s3Storage, compressor, crypter), nil

	default:
		return nil, fmt.Errorf("unimplemented repo type: %s", cfg.RepoType)
	}
}

func decideCompressorEncryptor(cfg *config.Config) (codec.Compressor, crypt.Crypter) {
	var compressor codec.Compressor
	var crypter crypt.Crypter

	if cfg.RepoCompressor != "" {
		slog.Info("init compressor",
			slog.String("module", "boot"),
			slog.String("compressor", string(cfg.RepoCompressor)),
		)

		switch cfg.RepoCompressor {
		case config.RepoCompressorGzip:
			compressor = &codec.GzipCompressor{}
		case config.RepoCompressorZstd:
			compressor = &codec.ZstdCompressor{}
		default:
			slog.Error("boot", "unknown-compression", cfg.RepoCompressor)
		}
	}
	if cfg.RepoEncryptor != "" {
		slog.Info("init crypter",
			slog.String("module", "boot"),
			slog.String("crypter", string(cfg.RepoEncryptor)),
		)

		if cfg.RepoEncryptor == config.RepoEncryptorAes256Gcm {
			crypter = aesgcm.NewChunkedGCMCrypter(cfg.RepoEncryptionPass)
		} else {
			slog.Error("boot", "unknown-encryption", cfg.RepoEncryptor)
		}
	}

	return compressor, crypter
}
