package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type s3Storage struct {
	client   *s3.Client
	bucket   string
	prefix   string
	uploader *manager.Uploader
}

var _ Storage = &s3Storage{}

func NewS3Storage(client *s3.Client, bucket, prefix string) Storage {
	return &s3Storage{
		client:   client,
		bucket:   bucket,
		prefix:   prefix,
		uploader: CreateUploader(client, 5242880, 2), // TODO:cfg
	}
}

// CreateUploader creates a new S3 uploader with the given part size and concurrency
func CreateUploader(client *s3.Client, partsize int64, concurrency int) *manager.Uploader {
	return manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = partsize
		u.Concurrency = concurrency
	})
}

func (s s3Storage) PutObject(ctx context.Context, path string, r io.Reader) error {
	path = s.fullPath(path)

	objInput := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   r,
	}

	_, err := s.uploader.Upload(ctx, objInput)
	if err != nil {
		return err
	}
	return nil
}

func (s s3Storage) ReadObject(ctx context.Context, path string) (io.ReadCloser, error) {
	path = s.fullPath(path)

	// TODO:design:fix: use *manager.Downloader

	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read object from S3: %w", err)
	}
	return out.Body, nil
}

func (s s3Storage) Exists(ctx context.Context, path string) (bool, error) {
	path = s.fullPath(path)

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		var nf *s3types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, err
	}
	return true, nil // S3 has no dirs, so it's a valid file
}

func (s s3Storage) SHA256(ctx context.Context, path string) (string, error) {
	path = s.fullPath(path)

	obj, err := s.ReadObject(ctx, path)
	if err != nil {
		return "", err
	}
	defer obj.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, obj); err != nil {
		return "", fmt.Errorf("failed to hash object: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s s3Storage) fullPath(path string) string {
	return filepath.ToSlash(filepath.Join(s.prefix, path))
}

func (s s3Storage) ListAll(ctx context.Context, prefix string) ([]string, error) {
	fullPath := s.fullPath(prefix)
	var objects []string

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(fullPath),
	})

	// Iterate over pages of results
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get page: %w", err)
		}

		for _, obj := range page.Contents {
			rel, err := filepath.Rel(s.prefix, *obj.Key)
			if err != nil {
				return nil, err
			}
			objects = append(objects, rel)
		}
	}

	return objects, nil
}

func (s s3Storage) ListTopLevelDirs(ctx context.Context, prefix string) (map[string]bool, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.bucket),
		Delimiter: aws.String("/"), // Groups results by prefix (like top-level directories)
		Prefix:    aws.String(prefix),
	}

	output, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects in bucket: %w", err)
	}

	// Extract top-level prefixes (directories)
	prefixes := make(map[string]bool)
	for _, prefix := range output.CommonPrefixes {
		if prefix.Prefix == nil {
			continue
		}
		prefixClean := strings.TrimSuffix(*prefix.Prefix, "/")
		rel, err := filepath.Rel(s.prefix, prefixClean)
		if err != nil {
			return nil, err
		}
		prefixes[filepath.ToSlash(rel)] = true
	}

	return prefixes, nil
}
