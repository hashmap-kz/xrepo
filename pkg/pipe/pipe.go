package pipe

import (
	"fmt"
	"io"

	"github.com/hashmap-kz/xrepo/pkg/codec"
	"github.com/hashmap-kz/xrepo/pkg/crypt"
)

// --- Pipeline ---

func CompressAndEncryptOptional(
	source io.Reader,
	compressor codec.Compressor,
	crypter crypt.Crypter,
) (io.Reader, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		var dst io.Writer = pw
		var encWriter io.WriteCloser
		var compWriter codec.WriteFlushCloser

		// Wrap encryption
		if crypter != nil {
			var err error
			encWriter, err = crypter.Encrypt(dst)
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			dst = encWriter
		}

		// Wrap compression
		if compressor != nil {
			var err error
			compWriter, err = compressor.NewWriter(dst)
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			dst = compWriter
		}

		// Copy source to top of stack (compressor or encryptor)
		if _, err := io.Copy(dst, source); err != nil {
			_ = pw.CloseWithError(fmt.Errorf("copy: %w", err))
			return
		}

		// Properly close in reverse order
		if compWriter != nil {
			_ = compWriter.Flush()
			_ = compWriter.Close()
		}
		if encWriter != nil {
			_ = encWriter.Close()
		}
	}()

	return pr, nil
}

func DecryptAndDecompressOptional(
	reader io.Reader,
	crypter crypt.Crypter,
	decompressor codec.Decompressor,
) (io.ReadCloser, error) {
	var err error

	// Decrypt
	if crypter != nil {
		reader, err = crypter.Decrypt(reader)
		if err != nil {
			return nil, err
		}
	}

	// If no decompression, wrap as ReadCloser if needed
	if decompressor == nil {
		if rc, ok := reader.(io.ReadCloser); ok {
			return rc, nil
		}
		return io.NopCloser(reader), nil
	}

	// Decompress
	return decompressor.Decompress(reader)
}
