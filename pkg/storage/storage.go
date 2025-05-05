package storage

import (
	"context"
	"io"
)

type Storage interface {
	PutObject(ctx context.Context, path string, r io.Reader) error

	ReadObject(ctx context.Context, path string) (io.ReadCloser, error)

	Exists(ctx context.Context, path string) (bool, error)

	SHA256(ctx context.Context, path string) (string, error)

	ListAll(ctx context.Context, prefix string) ([]string, error)

	ListTopLevelDirs(ctx context.Context, prefix string) (map[string]bool, error)
}
