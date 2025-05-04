package codec

import "io"

const (
	GzipFileExt = ".gz"
	ZstdFileExt = ".zst"
)

type Flusher interface {
	Flush() error
}

type WriteFlushCloser interface {
	io.WriteCloser
	Flusher
}

type Compressor interface {
	NewWriter(writer io.Writer) (WriteFlushCloser, error)
	FileExtension() string
	Name() string
}

type Decompressor interface {
	Decompress(src io.Reader) (io.ReadCloser, error)
	FileExtension() string
}

func GetDecompressor(c Compressor) Decompressor {
	if c == nil {
		return nil
	}
	ext := c.FileExtension()
	switch ext {
	case GzipFileExt:
		return &GzipDecompressor{}
	case ZstdFileExt:
		return &ZstdDecompressor{}
	default:
		return nil
	}
}
