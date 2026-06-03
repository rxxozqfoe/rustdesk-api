package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/lejianwen/rustdesk-api/v2/internal/config"
)

// Client wraps minio-go to provide S3-compatible storage operations.
type Client struct {
	mc     *minio.Client
	bucket string
}

// New creates an S3 client from config. Returns nil if S3 is not configured.
func New(cfg *config.S3) (*Client, error) {
	if !cfg.Enabled() {
		return nil, nil
	}

	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	return &Client{mc: mc, bucket: cfg.Bucket}, nil
}

// EnsureBucket creates the bucket if it does not exist.
func (c *Client) EnsureBucket(ctx context.Context, region string) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		if err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{Region: region}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	return nil
}

// UploadFile uploads a local file to the given S3 key. Returns the key on success.
func (c *Client) UploadFile(ctx context.Context, key, filePath, contentType string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	_, err = c.mc.PutObject(ctx, c.bucket, key, f, info.Size(), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3 key=%s: %w", key, err)
	}
	return key, nil
}

// UploadDir tars a directory and uploads as tar.gz to the given S3 key.
// For pre-build artifacts (directories), callers should tar.gz themselves
// and call UploadFile instead. This is a convenience for single-file uploads.
func (c *Client) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	_, err := c.mc.PutObject(ctx, c.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3 key=%s: %w", key, err)
	}
	return key, nil
}

// PresignedGetURL generates a presigned download URL valid for the given duration.
func (c *Client) PresignedGetURL(ctx context.Context, key string, expires time.Duration) (*url.URL, error) {
	reqParams := make(url.Values)
	return c.mc.PresignedGetObject(ctx, c.bucket, key, expires, reqParams)
}

// Delete removes an object from S3.
func (c *Client) Delete(ctx context.Context, key string) error {
	return c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}

// GetObject retrieves an object from S3 as a reader.
func (c *Client) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := c.mc.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 object key=%s: %w", key, err)
	}
	return obj, nil
}
