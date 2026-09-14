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
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	URLTTL    time.Duration
}

type MinIOStore struct {
	client *minio.Client
	bucket string
	ttl    time.Duration
}

func New(cfg Config) (*MinIOStore, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, errs.Wrap(errs.OpImageStoreConnect, err)
	}
	return &MinIOStore{client: client, bucket: cfg.Bucket, ttl: cfg.URLTTL}, nil
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
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, s.ttl, nil)
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
