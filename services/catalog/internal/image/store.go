// Package image puts product images in an S3-compatible object store. The
// same code reaches real S3 by changing only the endpoint, bucket and
// credentials in config — nothing here is provider-specific.
package image

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/loloDawit/go-admin/services/catalog/internal/errs"
)

// Store is the object-store boundary product code depends on; tests fake it, MinIOStore implements it for real.
type Store interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Presign(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}

// Config carries only what New needs. It is filled from services/catalog/internal/config, never from constants in this package.
type Config struct {
	Endpoint string
	// PublicEndpoint is the host a presigned URL names. It differs from
	// Endpoint whenever the service reaches the store over a private network a
	// browser cannot resolve, which is every containerised deployment.
	PublicEndpoint string
	Bucket         string
	AccessKey      string
	SecretKey      string
	UseSSL         bool
	URLTTL         time.Duration
}

const region = "us-east-1"

type MinIOStore struct {
	client  *minio.Client
	presign *minio.Client
	bucket  string
	ttl     time.Duration
}

func New(cfg Config) (*MinIOStore, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: region,
	})
	if err != nil {
		return nil, errs.Wrap(errs.OpImageStoreConnect, err)
	}
	presign := client
	if cfg.PublicEndpoint != "" && cfg.PublicEndpoint != cfg.Endpoint {
		// Region is set so the client never resolves the bucket's location:
		// that lookup would dial PublicEndpoint, which names a host reachable
		// from the browser and not from this process.
		presign, err = minio.New(cfg.PublicEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: cfg.UseSSL,
			Region: region,
		})
		if err != nil {
			return nil, errs.Wrap(errs.OpImageStoreConnect, err)
		}
	}
	return &MinIOStore{client: client, presign: presign, bucket: cfg.Bucket, ttl: cfg.URLTTL}, nil
}

func (s *MinIOStore) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return errs.Wrap(errs.OpImageStorePut, err)
	}
	return nil
}

// Presign's TTL is fixed at construction time from config, not passed per call: a per-call TTL would let a caller quietly widen the security control.
func (s *MinIOStore) Presign(ctx context.Context, key string) (string, error) {
	u, err := s.presign.PresignedGetObject(ctx, s.bucket, key, s.ttl, nil)
	if err != nil {
		return "", errs.Wrap(errs.OpImageStorePresign, err)
	}
	return u.String(), nil
}

func (s *MinIOStore) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return errs.Wrap(errs.OpImageStoreDelete, err)
	}
	return nil
}
