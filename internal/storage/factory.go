package storage

import (
	"context"
	"fmt"

	"yunt/internal/config"
)

// NewFromConfig creates a storage backend based on configuration.
func NewFromConfig(ctx context.Context, cfg config.StorageConfig) (Backend, error) {
	switch cfg.Type {
	case "filesystem":
		if cfg.Path == "" {
			return nil, fmt.Errorf("storage path is required for filesystem backend")
		}
		return NewFilesystemBackend(cfg.Path)
	case "s3":
		if cfg.S3Bucket == "" {
			return nil, fmt.Errorf("S3 bucket is required for S3 backend")
		}
		return NewS3Backend(ctx, S3Config{
			Bucket:    cfg.S3Bucket,
			Region:    cfg.S3Region,
			Prefix:    cfg.S3Prefix,
			Endpoint:  cfg.S3Endpoint,
			AccessKey: cfg.S3AccessKey,
			SecretKey: cfg.S3SecretKey,
		})
	case "db", "database", "":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}
