package lock

import (
	"context"
	"time"
)

// Locker is the minimal interface for a lock provider implementation.
// It must provide Lock (acquire) and Unlock (release) by key with a TTL and token verification.
type Locker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (string, error)
	Release(key string, token string) (bool, error)
}

type DistributedLock struct{}

func NewDistributedLock(provider Locker) Locker {
	return provider
}
