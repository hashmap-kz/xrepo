package repo

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	storage2 "github.com/hashmap-kz/xrepo/pkg/storage"

	"github.com/hashmap-kz/xrepo/pkg/crypt"
	"github.com/hashmap-kz/xrepo/pkg/crypt/aesgcm"

	"github.com/stretchr/testify/assert"

	"github.com/hashmap-kz/xrepo/pkg/codec"

	"github.com/stretchr/testify/require"
)

// --- Helpers ---

func readAllAndClose(t *testing.T, r io.ReadCloser) []byte {
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return data
}

func TestRepo_PutAndGet_GzipV1(t *testing.T) {
	tmp := t.TempDir()

	// Arrange
	local, err := storage2.NewLocal(tmp)
	require.NoError(t, err)

	gzipCompressor := &codec.GzipCompressor{}

	r := NewWriteReader(local, gzipCompressor, nil)

	content := []byte("This is a test backup payload.")
	logicalPath := "pg_data/PG_VERSION"

	// Act - store file
	path, err := r.PutObject(context.Background(), logicalPath, bytes.NewReader(content))
	require.NoError(t, err)

	// Assert - file physically exists
	_, err = os.Stat(filepath.Join(tmp, path))
	require.NoError(t, err)

	// Act - retrieve and decompress file
	reader, err := r.ReadObject(context.Background(), logicalPath)
	require.NoError(t, err)

	result, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.NoError(t, reader.Close())
	assert.Equal(t, content, result)
}

func TestRepo_PutAndGet_NoCompression_NoEncryption(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage2.NewLocal(tmp)
	require.NoError(t, err)

	r := NewWriteReader(store, nil, nil)

	content := []byte("raw test data")
	logicalPath := "plain/data.txt"

	// Write
	finalPath, err := r.PutObject(context.Background(), logicalPath, bytes.NewReader(content))
	require.NoError(t, err)

	// Read
	rc, err := r.ReadObject(context.Background(), logicalPath)
	require.NoError(t, err)

	result := readAllAndClose(t, rc)
	assert.Equal(t, content, result)

	// Check physical file
	_, err = os.Stat(filepath.Join(tmp, finalPath))
	assert.NoError(t, err)
}

func TestRepo_PutAndGet_GzipV2(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage2.NewLocal(tmp)
	require.NoError(t, err)

	r := NewWriteReader(store, &codec.GzipCompressor{}, nil)

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

func TestRepo_PutAndGet_EncryptionOnly(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage2.NewLocal(tmp)
	require.NoError(t, err)

	crypter := aesgcm.NewChunkedGCMCrypter("secret-key") // assume implementation exists

	r := NewWriteReader(store, nil, crypter)

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

func TestRepo_PutAndGet_GzipAndEncryption(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage2.NewLocal(tmp)
	require.NoError(t, err)

	crypter := aesgcm.NewChunkedGCMCrypter("combo-key")
	compressor := &codec.GzipCompressor{}

	r := NewWriteReader(store, compressor, crypter)

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

func TestRepo_EmptyFile(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage2.NewLocal(tmp)
	require.NoError(t, err)

	r := NewWriteReader(store, &codec.GzipCompressor{}, nil)

	var empty bytes.Buffer

	finalPath, err := r.PutObject(context.Background(), "empty/file.txt", &empty)
	require.NoError(t, err)

	rc, err := r.ReadObject(context.Background(), "empty/file.txt")
	require.NoError(t, err)

	result := readAllAndClose(t, rc)
	assert.Empty(t, result)

	assert.True(t, strings.HasSuffix(finalPath, ".gz"))
}

func TestRepo_InvalidCompressorExtension(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage2.NewLocal(tmp)
	require.NoError(t, err)

	// fake compressor without matching decompressor
	fake := fakeCompressor{}
	r := NewWriteReader(store, fake, nil)

	_, err = r.PutObject(context.Background(), "broken/file", bytes.NewReader([]byte("xxx")))
	require.NoError(t, err)

	_, err = r.ReadObject(context.Background(), "broken/file")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot decide decompressor")
}

func TestRepo_ReadObject_DecompressFails(t *testing.T) {
	store := &mockStorage{
		files: map[string][]byte{
			"test.gz": []byte("invalid gzip data"),
		},
	}

	compressor := &codec.GzipCompressor{}
	r := NewWriteReader(store, compressor, nil)

	_, err := r.ReadObject(context.Background(), "test")
	assert.Error(t, err)
	assert.True(t, store.closed["test.gz"], "expected obj.Close() to be called on decompress error")
}

func TestRepo_BadReaderWrite(t *testing.T) {
	tmp := t.TempDir()
	store, err := storage2.NewLocal(tmp)
	require.NoError(t, err)

	r := NewWriteReader(store, nil, nil)

	badReader := &failReader{}
	_, err = r.PutObject(context.Background(), "bad/path", badReader)
	assert.Error(t, err)
}

func TestEncodePath(t *testing.T) {
	tests := []struct {
		name       string
		logical    string
		compressor codec.Compressor
		crypter    crypt.Crypter
		expected   string
	}{
		{
			name:     "no compressor or crypter",
			logical:  "pg_data/base/123",
			expected: filepath.ToSlash("pg_data/base/123"),
		},
		{
			name:       "gzip only",
			logical:    "pg_data/base/123",
			compressor: newMockCompressor(".gz"),
			expected:   filepath.ToSlash("pg_data/base/123.gz"),
		},
		{
			name:     "aes only",
			logical:  "pg_data/base/123",
			crypter:  newMockCrypter(".aes"),
			expected: filepath.ToSlash("pg_data/base/123.aes"),
		},
		{
			name:       "gzip + aes",
			logical:    "pg_data/base/123",
			compressor: newMockCompressor(".gz"),
			crypter:    newMockCrypter(".aes"),
			expected:   filepath.ToSlash("pg_data/base/123.gz.aes"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &repoImpl{
				compressor: tt.compressor,
				crypter:    tt.crypter,
			}
			result := r.encodePath(tt.logical)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKnownExtensions(t *testing.T) {
	r := &repoImpl{
		compressor: newMockCompressor(".zst"),
		crypter:    newMockCrypter(".enc"),
	}

	got := r.knownExtensions()

	expected := []string{".zst.enc", ".zst", ".enc"}
	sort.Strings(expected)
	sort.Strings(got)

	assert.ElementsMatch(t, expected, got)
}

func TestDecodePath(t *testing.T) {
	r := &repoImpl{}
	exts := []string{".zst.enc", ".zst", ".enc"}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with .zst.enc", "foo/bar/file.zst.enc", "foo/bar/file"},
		{"with .zst", "file.zst", "file"},
		{"with .enc", "file.enc", "file"},
		{"no extension", "pg_data/base/123", "pg_data/base/123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded := r.decodePath(tt.input, exts)
			assert.Equal(t, filepath.ToSlash(tt.expected), decoded)
		})
	}
}

// --- Mocks ---

type failReader struct{}

func (f *failReader) Read([]byte) (int, error) {
	return 0, errors.New("read fail")
}

// storage

type mockCloserReader struct {
	io.Reader
	onClose func()
}

func (m *mockCloserReader) Close() error {
	if m.onClose != nil {
		m.onClose()
	}
	return nil
}

type mockStorage struct {
	files  map[string][]byte
	closed map[string]bool
}

func (m *mockStorage) ListAll(_ context.Context, _ string) ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (m *mockStorage) ListTopLevelDirs(_ context.Context, _ string) (map[string]bool, error) {
	// TODO implement me
	panic("implement me")
}

var _ storage2.Storage = &mockStorage{}

func (m *mockStorage) PutObject(_ context.Context, _ string, _ io.Reader) error {
	return nil
}

func (m *mockStorage) ReadObject(_ context.Context, path string) (io.ReadCloser, error) {
	if m.closed == nil {
		m.closed = make(map[string]bool)
	}
	data, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &mockCloserReader{
		Reader: bytes.NewReader(data),
		onClose: func() {
			m.closed[path] = true
		},
	}, nil
}

func (m *mockStorage) Exists(_ context.Context, _ string) (bool, error)   { return true, nil }
func (m *mockStorage) SHA256(_ context.Context, _ string) (string, error) { return "", nil }

// compressor

type fakeCompressor struct{}

func (c fakeCompressor) Name() string {
	return "fakeCompressor"
}

var _ codec.Compressor = &fakeCompressor{}

func (fakeCompressor) FileExtension() string { return ".fake" }
func (fakeCompressor) NewWriter(w io.Writer) (codec.WriteFlushCloser, error) {
	return nopCloser{w}, nil
}

type nopCloser struct{ io.Writer }

func (n nopCloser) Flush() error { return nil }
func (n nopCloser) Close() error { return nil }

// fakes with extensions set

type mockCompressor struct {
	ext string
}

func (m *mockCompressor) Name() string {
	return "mockCompressor"
}

func newMockCompressor(ext string) codec.Compressor {
	return &mockCompressor{ext: ext}
}

func (m *mockCompressor) NewWriter(_ io.Writer) (codec.WriteFlushCloser, error) { return nil, nil }
func (m *mockCompressor) Decompress(_ io.Reader) (io.ReadCloser, error)         { return nil, nil }
func (m *mockCompressor) FileExtension() string                                 { return m.ext }

type mockCrypter struct {
	ext string
}

func newMockCrypter(ext string) crypt.Crypter {
	return &mockCrypter{ext: ext}
}

func (m *mockCrypter) Encrypt(_ io.Writer) (io.WriteCloser, error) { return nil, nil }
func (m *mockCrypter) Decrypt(_ io.Reader) (io.Reader, error)      { return nil, nil }
func (m *mockCrypter) FileExtension() string                       { return m.ext }
func (m *mockCrypter) Name() string                                { return "" }
