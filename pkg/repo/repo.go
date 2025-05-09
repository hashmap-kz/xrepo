package repo

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/hashmap-kz/streamcrypt/pkg/ioutils"
	"github.com/hashmap-kz/streamcrypt/pkg/pipe"
	"github.com/hashmap-kz/xrepo/pkg/storage"

	"github.com/hashmap-kz/streamcrypt/pkg/codec"
	"github.com/hashmap-kz/streamcrypt/pkg/crypt"
)

type WriteReader interface {
	// PutObject writes file to storage (through compress/encrypt pipes), returns final name (with optional extensions: *.gz, *.gz.aes)
	PutObject(ctx context.Context, path string, r io.Reader) (string, error)

	// PutObjectPlain saves object without applying compression/encryption (i.e: manifest writing for debug, etc...)
	PutObjectPlain(ctx context.Context, path string, r io.Reader) (string, error)

	ReadObject(ctx context.Context, path string) (io.ReadCloser, error)

	Exists(ctx context.Context, path string) (bool, error)

	ListAll(ctx context.Context, prefix string) ([]string, error)

	ListTopLevelDirs(ctx context.Context, prefix string) (map[string]bool, error)

	GetCompressorName() string

	GetEncryptorName() string
}

type repoImpl struct {
	storage    storage.Storage  // required: e.g. LocalImpl()
	compressor codec.Compressor // optional
	crypter    crypt.Crypter    // optional
}

var _ WriteReader = &repoImpl{}

func NewWriteReader(s storage.Storage, compressor codec.Compressor, crypter crypt.Crypter) WriteReader {
	return &repoImpl{
		storage:    s,
		compressor: compressor,
		crypter:    crypter,
	}
}

func (repo *repoImpl) PutObject(ctx context.Context, path string, r io.Reader) (string, error) {
	var err error
	fullPath := repo.encodePath(path)

	// Compress and encrypt
	encReader, err := pipe.CompressAndEncryptOptional(r, repo.compressor, repo.crypter)
	if err != nil {
		return "", err
	}

	// Store in repo
	err = repo.storage.PutObject(ctx, fullPath, encReader)
	if err != nil {
		return "", err
	}

	return fullPath, nil
}

func (repo *repoImpl) PutObjectPlain(ctx context.Context, path string, r io.Reader) (string, error) {
	var err error
	fullPath := filepath.ToSlash(path)

	// Store in repo
	err = repo.storage.PutObject(ctx, fullPath, r)
	if err != nil {
		return "", err
	}

	return fullPath, nil
}

func (repo *repoImpl) ReadObject(ctx context.Context, path string) (io.ReadCloser, error) {
	var err error
	fullPath := repo.encodePath(path)

	// Open() that needs to be closed
	obj, err := repo.storage.ReadObject(ctx, fullPath)
	if err != nil {
		return nil, err
	}

	var dec codec.Decompressor
	if repo.compressor != nil {
		dec = codec.GetDecompressor(repo.compressor)
		if dec == nil {
			obj.Close()
			return nil, fmt.Errorf("cannot decide decompressor for: %s", repo.compressor.FileExtension())
		}
	}

	readCloser, err := pipe.DecryptAndDecompressOptional(obj, repo.crypter, dec)
	if err != nil {
		obj.Close()
		return nil, err
	}

	return ioutils.NewMultiCloser(readCloser, obj, readCloser), nil
}

func (repo *repoImpl) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := repo.encodePath(path)
	return repo.storage.Exists(ctx, fullPath)
}

func (repo *repoImpl) ListAll(ctx context.Context, prefix string) ([]string, error) {
	// objects in storage are saved with optional extensions: *.gz, *.gz.aes, etc...
	// but repo is working ONLY with plain names, and handles compression/encryption
	// so: we need to trim extensions and return a clean list of names
	storageObjects, err := repo.storage.ListAll(ctx, prefix)
	if err != nil {
		return nil, err
	}

	// no compression, no encryption, return as is
	if repo.compressor == nil && repo.crypter == nil {
		return storageObjects, nil
	}

	// trim all possible extensions
	filtered := make([]string, 0, len(storageObjects))
	for _, elem := range storageObjects {
		cleaned := repo.decodePath(elem)
		filtered = append(filtered, filepath.ToSlash(cleaned))
	}
	return filtered, nil
}

func (repo *repoImpl) ListTopLevelDirs(ctx context.Context, prefix string) (map[string]bool, error) {
	return repo.storage.ListTopLevelDirs(ctx, prefix)
}

// path-utils

// encodePath adds extensions based on active compressor/crypter
func (repo *repoImpl) encodePath(logical string) string {
	ext := ""
	if repo.compressor != nil {
		ext += repo.compressor.FileExtension()
	}
	if repo.crypter != nil {
		ext += repo.crypter.FileExtension()
	}
	return filepath.ToSlash(logical + ext)
}

// decodePath removes known extensions
func (repo *repoImpl) decodePath(path string) string {
	ext := extractCompoundExt(path)
	if ext == "" {
		return path
	}
	return strings.TrimSuffix(path, ext)
}

// ExtractCompoundExt returns the extension starting from the first dot,
// e.g., ".gz.aes", ".tar.gz.aes", or "" if no dot.
func extractCompoundExt(name string) string {
	name = filepath.Base(name)
	i := strings.IndexByte(name, '.')
	if i < 0 {
		return "" // no extension
	}
	return name[i:] // includes the dot
}

func (repo *repoImpl) GetCompressorName() string {
	if repo.compressor != nil {
		return repo.compressor.Name()
	}
	return ""
}

func (repo *repoImpl) GetEncryptorName() string {
	if repo.crypter != nil {
		return repo.crypter.Name()
	}
	return ""
}
