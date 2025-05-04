//go:build integration

package integration

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	storage2 "github.com/hashmap-kz/xrepo/pkg/storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func storageTestsCreateS3Client(t *testing.T) (*s3.Client, storage2.Storage) {
	t.Helper()
	return createS3Client("storage-unittest")
}

func TestS3Storage_ListAll(t *testing.T) {
	_, store := storageTestsCreateS3Client(t)

	ctx := context.Background()
	_ = store.PutObject(ctx, "listall/a.txt", bytes.NewReader([]byte("A")))
	_ = store.PutObject(ctx, "listall/b.txt", bytes.NewReader([]byte("B")))

	time.Sleep(500 * time.Millisecond) // MinIO consistency delay

	files, err := store.ListAll(ctx, "listall/")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"listall/a.txt", "listall/b.txt"}, files)
}

func TestS3Storage_ListTopLevelDirs(t *testing.T) {
	ctx := context.Background()
	client, _ := storageTestsCreateS3Client(t)
	bucket := "unittest-bucket"
	prefix := "test-prefix/"

	// Create bucket if it doesn't exist
	_, _ = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})

	// Clean up from previous runs
	list, _ := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	for _, obj := range list.Contents {
		_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    obj.Key,
		})
	}

	// Upload test objects in different "dirs"
	keys := []string{
		prefix + "dir1/file1.txt",
		prefix + "dir2/file2.txt",
		prefix + "dir2/file3.txt",
		prefix + "dir3/subdir/file.txt",
	}
	for _, key := range keys {
		_, err := client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   strings.NewReader("hello"),
		})
		assert.NoError(t, err)
	}

	s := storage2.NewS3Storage(client, bucket, prefix)
	dirs, err := s.ListTopLevelDirs(ctx, prefix)
	assert.NoError(t, err)

	// Verify detected top-level dirs
	assert.True(t, dirs["dir1"])
	assert.True(t, dirs["dir2"])
	assert.True(t, dirs["dir3"])
	assert.Len(t, dirs, 3)
}
