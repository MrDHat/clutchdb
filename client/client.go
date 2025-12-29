package client

import (
	"context"
	"time"
)

type Client interface {
	Acquire(ctx context.Context, lock string, ttl time.Duration) (uint64, error)
	Renew(ctx context.Context, lock string, ttl time.Duration) (uint64, error)
	Release(ctx context.Context, lock string) error
}
