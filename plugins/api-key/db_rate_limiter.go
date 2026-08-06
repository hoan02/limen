package apikey

import (
	"context"
	"fmt"
	"time"

	"github.com/thecodearcher/limen"
)

type rateLimiter interface {
	Enforce(ctx context.Context, apiKey *ApiKey) error
}

type databaseRateLimiter struct {
	self         *apiKeyPlugin
	core         *limen.LimenCore
	apiKeySchema *apiKeySchema
}

func newDatabaseRateLimiter(plugin *apiKeyPlugin) *databaseRateLimiter {
	return &databaseRateLimiter{self: plugin, core: plugin.core, apiKeySchema: plugin.apiKeySchema}
}

func (r *databaseRateLimiter) Enforce(ctx context.Context, apiKey *ApiKey) error {
	return r.applyRateLimit(ctx, apiKey)
}

func (r *databaseRateLimiter) checkRateLimit(apiKey *ApiKey) rateLimitAction {
	if !apiKey.RateLimitEnabled() {
		return rateLimitTouch
	}

	if apiKey.LastUsedAt == nil || apiKey.RateLimitRequestCount == nil {
		return rateLimitReset
	}

	idleTimeout := time.Duration(*apiKey.RateLimitWindowMS) * time.Millisecond
	if time.Since(*apiKey.LastUsedAt) >= idleTimeout {
		return rateLimitReset
	}

	if *apiKey.RateLimitRequestCount >= *apiKey.RateLimitMax {
		return rateLimitReject
	}

	return rateLimitIncrement
}

func (r *databaseRateLimiter) applyRateLimit(ctx context.Context, apiKey *ApiKey) error {
	action := r.checkRateLimit(apiKey)
	switch action {
	case rateLimitReject:
		return ErrRateLimitExceeded
	case rateLimitTouch:
		return r.touchLastUsedAt(ctx, apiKey)
	case rateLimitReset:
		won, err := r.resetCounterIfUnchanged(ctx, apiKey)
		if err != nil {
			return err
		}
		if won {
			return nil
		}
		return r.incrementUnderLimit(ctx, apiKey)
	case rateLimitIncrement:
		return r.incrementUnderLimit(ctx, apiKey)
	default:
		return fmt.Errorf("unknown rate-limit action: %d", action)
	}
}

func (r *databaseRateLimiter) touchLastUsedAt(ctx context.Context, apiKey *ApiKey) error {
	return r.self.store.Update(ctx, apiKey, map[limen.SchemaField]any{APIKeySchemaLastUsedAtField: time.Now()}, []limen.Where{
		limen.Eq(r.self.apiKeySchema.GetIDField(), apiKey.ID),
	})
}

// resetCounterIfUnchanged starts a new activity period only if the last-used
// value has not changed since it was read.
func (r *databaseRateLimiter) resetCounterIfUnchanged(ctx context.Context, apiKey *ApiKey) (bool, error) {
	var anchorGuard limen.Where
	if apiKey.LastUsedAt == nil {
		anchorGuard = limen.IsNull(r.apiKeySchema.GetLastUsedAtField())
	} else {
		anchorGuard = limen.Eq(r.apiKeySchema.GetLastUsedAtField(), *apiKey.LastUsedAt)
	}

	res, err := r.core.UpdateWithResult(ctx, r.apiKeySchema, map[limen.SchemaField]any{
		APIKeySchemaRateLimitRequestCountField: int32(1),
		APIKeySchemaLastUsedAtField:            time.Now(),
	}, []limen.Where{
		limen.Eq(r.apiKeySchema.GetIDField(), apiKey.ID),
		anchorGuard,
	})
	if err != nil {
		return false, err
	}
	return res.RowsAffected == 1, nil
}

// incrementUnderLimit atomically increments the counter while capacity remains.
func (r *databaseRateLimiter) incrementUnderLimit(ctx context.Context, apiKey *ApiKey) error {
	res, err := r.core.UpdateWithResult(ctx, r.apiKeySchema, map[limen.SchemaField]any{
		APIKeySchemaRateLimitRequestCountField: limen.IncrementBy(1),
		APIKeySchemaLastUsedAtField:            time.Now(),
	}, []limen.Where{
		limen.Eq(r.apiKeySchema.GetIDField(), apiKey.ID),
		limen.Lt(r.apiKeySchema.GetRateLimitRequestCountField(), *apiKey.RateLimitMax),
	})
	if err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return ErrRateLimitExceeded
	}
	return nil
}
