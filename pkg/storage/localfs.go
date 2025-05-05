package storage

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hashmap-kz/xrepo/pkg/fsync"

	"github.com/hashmap-kz/xrepo/pkg/common"
)

type localStorage struct {
	baseDir      string
	fsyncOnWrite bool
}

type LocalStorageOpts struct {
	BaseDir      string
	FsyncOnWrite bool
}

var _ Storage = &localStorage{}

func NewLocal(o *LocalStorageOpts) (Storage, error) {
	if err := os.MkdirAll(o.BaseDir, 0o750); err != nil {
		return nil, err
	}
	return &localStorage{baseDir: o.BaseDir, fsyncOnWrite: o.FsyncOnWrite}, nil
}

func (l *localStorage) fullPath(path string) string {
	return filepath.ToSlash(filepath.Join(l.baseDir, filepath.Clean(path)))
}

func (l *localStorage) PutObject(_ context.Context, path string, r io.Reader) error {
	fullPath := l.fullPath(path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
		return err
	}
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}

	// Copy contents
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close() // ignore close error if we already have a copy error
		return err
	}

	// Fsync if needed
	if l.fsyncOnWrite {
		if err := fsync.Fsync(f); err != nil {
			_ = f.Close() // same here: best-effort
			return err
		}
	}

	// Now close, and return any close error
	return f.Close()
}

func (l *localStorage) ReadObject(_ context.Context, path string) (io.ReadCloser, error) {
	return os.Open(l.fullPath(path))
}

func (l *localStorage) Exists(_ context.Context, path string) (bool, error) {
	fullPath := l.fullPath(path)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if info.Mode().IsRegular() {
		return true, nil
	}
	return false, nil
}

func (l *localStorage) SHA256(_ context.Context, path string) (string, error) {
	fullPath := l.fullPath(path)
	return common.Sha256FromFile(fullPath)
}

func (l *localStorage) ListAll(_ context.Context, prefix string) ([]string, error) {
	fullPath := l.fullPath(prefix)
	var result []string

	err := filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %q: %w", path, err)
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(l.baseDir, path)
		if err != nil {
			return err
		}
		result = append(result, filepath.ToSlash(rel))
		return nil
	})
	return result, err
}

func (l *localStorage) ListTopLevelDirs(_ context.Context, prefix string) (map[string]bool, error) {
	result := make(map[string]bool)

	entries, err := os.ReadDir(prefix)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			fullPath := filepath.ToSlash(filepath.Join(prefix, entry.Name()))
			rel, err := filepath.Rel(l.baseDir, fullPath)
			if err != nil {
				return nil, err
			}
			result[filepath.ToSlash(rel)] = true
		}
	}
	return result, nil
}
