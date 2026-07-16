package apikey

import (
	"context"
	"fmt"
	"time"

	"github.com/thecodearcher/limen"
)

type cacheRateLimiter struct {
	self   *apiKeyPlugin
	prefix string
	cache  limen.AtomicCacheAdapter
}

func newCacheRateLimiter(plugin *apiKeyPlugin) *cacheRateLimiter {
	return &cacheRateLimiter{
		self:   plugin,
		prefix: plugin.core.CacheKeyPrefix(),
		cache:  plugin.core.AtomicCacheStore(),
	}
}

func (r *cacheRateLimiter) key(id any) string {
	return fmt.Sprintf("%s:api-key:rl:%v", r.prefix, id)
}

func (r *cacheRateLimiter) Enforce(ctx context.Context, apiKey *ApiKey) error {
	if !apiKey.RateLimitEnabled() {
		return nil
	}

	key := r.key(apiKey.ID)
	window := time.Duration(*apiKey.RateLimitWindowMS) * time.Millisecond

	newCount, err := r.cache.Increment(ctx, key, 1)
	if err != nil {
		return err
	}

	if newCount > int64(*apiKey.RateLimitMax) {
		return ErrRateLimitExceeded
	}

	if err := r.cache.SetExpiry(ctx, key, window); err != nil {
		return err
	}

	if err := r.touchLastUsedAt(ctx, apiKey); err != nil {
		return err
	}

	return nil
}

func (r *cacheRateLimiter) touchLastUsedAt(ctx context.Context, apiKey *ApiKey) error {
	if apiKey.LastUsedAt != nil && time.Since(*apiKey.LastUsedAt) < r.self.config.lastUsedAtThrottle {
		return nil
	}

	return r.self.store.Update(ctx, apiKey, map[limen.SchemaField]any{APIKeySchemaLastUsedAtField: time.Now()}, []limen.Where{
		limen.Eq(r.self.apiKeySchema.GetIDField(), apiKey.ID),
	})
}
