// Package redis provides a Redis-backed cache adapter for Limen.
package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/thecodearcher/limen"
)

var (
	_ limen.CacheAdapter       = (*Adapter)(nil)
	_ limen.AtomicCacheAdapter = (*Adapter)(nil)
)

type Adapter struct {
	client goredis.UniversalClient
}

func New(client goredis.UniversalClient) *Adapter {
	return &Adapter{client: client}
}

func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := a.client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, limen.ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (a *Adapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl < 0 {
		ttl = 0
	}
	return a.client.Set(ctx, key, value, ttl).Err()
}

func (a *Adapter) Has(ctx context.Context, key string) (bool, error) {
	count, err := a.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (a *Adapter) Delete(ctx context.Context, key string) error {
	return a.client.Del(ctx, key).Err()
}

// SetExpiry updates a value's expiry. A non-positive TTL removes its expiry.
func (a *Adapter) SetExpiry(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return a.client.Persist(ctx, key).Err()
	}
	return a.client.Expire(ctx, key, ttl).Err()
}

func (a *Adapter) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	return a.client.IncrBy(ctx, key, delta).Result()
}

func (a *Adapter) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	return a.client.DecrBy(ctx, key, delta).Result()
}
