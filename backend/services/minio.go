package services

import (
	"context"
	"fmt"
	"io"

	"github.com/mathornton01/arkheion/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
)

// MinIOService wraps the MinIO client for file storage operations.
type MinIOService struct {
	cfg    *config.Config
	client *minio.Client
}

// NewMinIOService creates a new MinIOService and ensures the configured bucket exists.
func NewMinIOService(cfg *config.Config) (*MinIOService, error) {
	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create MinIO client: %w", err)
	}

	svc := &MinIOService{cfg: cfg, client: client}

	// Ensure bucket exists
	if err := svc.ensureBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure MinIO bucket: %w", err)
	}

	return svc, nil
}

// ensureBucket creates the configured bucket if it doesn't already exist.
func (s *MinIOService) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.cfg.MinioBucket)
	if err != nil {
		return fmt.Errorf("check bucket existence: %w", err)
	}
	if exists {
		log.Debug().Str("bucket", s.cfg.MinioBucket).Msg("MinIO bucket already exists")
		return nil
	}

	if err := s.client.MakeBucket(ctx, s.cfg.MinioBucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create bucket %q: %w", s.cfg.MinioBucket, err)
	}
	log.Info().Str("bucket", s.cfg.MinioBucket).Msg("MinIO bucket created")
	return nil
}

// UploadFile streams content to MinIO and returns the number of bytes written.
// objectKey is the full path within the bucket (e.g. "books/{id}/file.pdf").
func (s *MinIOService) UploadFile(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) (int64, error) {
	info, err := s.client.PutObject(ctx, s.cfg.MinioBucket, objectKey, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return 0, fmt.Errorf("MinIO PutObject %q: %w", objectKey, err)
	}

	log.Info().
		Str("bucket", s.cfg.MinioBucket).
		Str("key", objectKey).
		Int64("size", info.Size).
		Msg("File uploaded to MinIO")

	return info.Size, nil
}

// DownloadFile retrieves a file from MinIO and returns a ReadCloser.
// The caller MUST close the returned reader.
// Returns the object size and content type for HTTP response headers.
func (s *MinIOService) DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, int64, string, error) {
	obj, err := s.client.GetObject(ctx, s.cfg.MinioBucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, "", fmt.Errorf("MinIO GetObject %q: %w", objectKey, err)
	}

	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, 0, "", fmt.Errorf("MinIO Stat %q: %w", objectKey, err)
	}

	return obj, info.Size, info.ContentType, nil
}

// DeleteFile removes an object from MinIO.
func (s *MinIOService) DeleteFile(objectKey string) error {
	err := s.client.RemoveObject(context.Background(), s.cfg.MinioBucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("MinIO RemoveObject %q: %w", objectKey, err)
	}
	log.Info().Str("key", objectKey).Msg("File deleted from MinIO")
	return nil
}

// GetPresignedURL generates a pre-signed URL for direct browser access to an object.
// This is NOT used for book file downloads (those go through the backend for auth),
// but may be used for cover images if configured for public access.
func (s *MinIOService) GetPresignedURL(ctx context.Context, objectKey string) (string, error) {
	// For public MinIO, construct the URL directly
	return fmt.Sprintf("%s/%s/%s", s.cfg.MinioPublicURL, s.cfg.MinioBucket, objectKey), nil
}
