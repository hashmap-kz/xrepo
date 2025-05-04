package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
)

type sftpStorage struct {
	client *sftp.Client
	root   string
}

var _ Storage = &sftpStorage{}

func NewSFTPStorage(client *sftp.Client, remoteDir string) Storage {
	return &sftpStorage{
		client: client,
		root:   strings.TrimSuffix(remoteDir, "/"),
	}
}

func (s *sftpStorage) resolvePath(p string) string {
	return filepath.ToSlash(path.Join(s.root, p))
}

func (s *sftpStorage) PutObject(_ context.Context, relPath string, r io.Reader) error {
	fullPath := s.resolvePath(relPath)

	// Ensure directory exists
	dir := path.Dir(fullPath)
	if err := s.client.MkdirAll(dir); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Open file for writing
	f, err := s.client.Create(fullPath)
	if err != nil {
		return fmt.Errorf("sftp create: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}

func (s *sftpStorage) ReadObject(_ context.Context, relPath string) (io.ReadCloser, error) {
	fullPath := s.resolvePath(relPath)
	f, err := s.client.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("sftp open: %w", err)
	}
	return f, nil
}

func (s *sftpStorage) Exists(_ context.Context, relPath string) (bool, error) {
	fullPath := s.resolvePath(relPath)
	info, err := s.client.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.Mode().IsRegular(), nil
}

func (s *sftpStorage) SHA256(ctx context.Context, relPath string) (string, error) {
	rc, err := s.ReadObject(ctx, relPath)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	h := sha256.New()
	if _, err := io.Copy(h, rc); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *sftpStorage) ListAll(_ context.Context, prefix string) ([]string, error) {
	fullPath := s.fullPath(prefix)
	var result []string

	walker := s.client.Walk(fullPath)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return nil, fmt.Errorf("error walking directory: %w", err)
		}
		stat := walker.Stat()
		if stat == nil {
			continue
		}
		if stat.IsDir() {
			continue
		}
		if walker.Path() != fullPath {
			rel, err := filepath.Rel(s.root, walker.Path())
			if err != nil {
				return nil, err
			}
			result = append(result, rel)
		}
	}

	return result, nil
}

func (s *sftpStorage) ListTopLevelDirs(_ context.Context, prefix string) (map[string]bool, error) {
	result := make(map[string]bool)

	entries, err := s.client.ReadDir(prefix)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			fullPath := filepath.ToSlash(filepath.Join(prefix, entry.Name()))
			rel, err := filepath.Rel(s.root, fullPath)
			if err != nil {
				return nil, err
			}
			result[filepath.ToSlash(rel)] = true
		}
	}
	return result, nil
}

func (s *sftpStorage) fullPath(p string) string {
	return filepath.ToSlash(filepath.Join(s.root, filepath.Clean(p)))
}
