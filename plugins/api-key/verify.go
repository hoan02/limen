package apikey

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func (p *apiKeyPlugin) Verify(ctx context.Context, key string, opts *VerifyOptions) (*ApiKey, error) {
	keyHash := p.hashAPIKey(key)

	apiKeyModel, err := p.core.FindOne(ctx, p.apiKeySchema, []limen.Where{
		limen.Eq(p.apiKeySchema.GetKeyHashField(), keyHash),
	}, nil)

	if err != nil {
		if errors.Is(err, limen.ErrRecordNotFound) {
			return nil, ErrInvalidAPIKey
		}
		return nil, err
	}

	apiKey := apiKeyModel.(*ApiKey)
	profile, err := p.resolveVerificationProfile(apiKey, opts.ProfileID)
	if err != nil {
		return nil, err
	}

	if !apiKey.Enabled {
		return nil, ErrAPIKeyDisabled
	}

	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, ErrAPIKeyExpired
	}

	if profile.KeyVerifier != nil && !profile.KeyVerifier(key) {
		return nil, ErrInvalidAPIKey
	}

	if err := p.applyRateLimit(ctx, apiKey); err != nil {
		return nil, err
	}

	if opts.RequiredPermissions != nil {
		if !access.HasRequiredPermissions(apiKey.Permissions, opts.RequiredPermissions) {
			return nil, ErrInsufficientPermissions
		}
	}

	return apiKey, nil
}

func (p *apiKeyPlugin) resolveVerificationProfile(apiKey *ApiKey, specifiedProfileID string) (*Profile, error) {
	if specifiedProfileID != "" && specifiedProfileID != apiKey.Profile {
		return nil, ErrInvalidAPIKey
	}
	return p.GetProfile(apiKey.Profile)
}

func (p *apiKeyPlugin) checkRateLimit(apiKey *ApiKey) rateLimitAction {
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

func (p *apiKeyPlugin) applyRateLimit(ctx context.Context, apiKey *ApiKey) error {
	action := p.checkRateLimit(apiKey)
	switch action {
	case rateLimitReject:
		return ErrRateLimitExceeded
	case rateLimitTouch:
		return p.touchLastUsedAt(ctx, apiKey.ID)
	case rateLimitReset:
		won, err := p.resetCounterIfUnchanged(ctx, apiKey)
		if err != nil {
			return err
		}
		if won {
			return nil
		}
		return p.incrementUnderLimit(ctx, apiKey)
	case rateLimitIncrement:
		return p.incrementUnderLimit(ctx, apiKey)
	default:
		return fmt.Errorf("unknown rate-limit action: %d", action)
	}
}

func (p *apiKeyPlugin) touchLastUsedAt(ctx context.Context, apiKeyID any) error {
	return p.core.Update(ctx, p.apiKeySchema, map[limen.SchemaField]any{
		APIKeySchemaLastUsedAtField: time.Now(),
	}, []limen.Where{
		limen.Eq(p.apiKeySchema.GetIDField(), apiKeyID),
	})
}

// resetCounterIfUnchanged starts a new activity period only if the last-used
// value has not changed since it was read.
func (p *apiKeyPlugin) resetCounterIfUnchanged(ctx context.Context, apiKey *ApiKey) (bool, error) {
	var anchorGuard limen.Where
	if apiKey.LastUsedAt == nil {
		anchorGuard = limen.IsNull(p.apiKeySchema.GetLastUsedAtField())
	} else {
		anchorGuard = limen.Eq(p.apiKeySchema.GetLastUsedAtField(), *apiKey.LastUsedAt)
	}

	res, err := p.core.UpdateWithResult(ctx, p.apiKeySchema, map[limen.SchemaField]any{
		APIKeySchemaRateLimitRequestCountField: int32(1),
		APIKeySchemaLastUsedAtField:            time.Now(),
	}, []limen.Where{
		limen.Eq(p.apiKeySchema.GetIDField(), apiKey.ID),
		anchorGuard,
	})
	if err != nil {
		return false, err
	}
	return res.RowsAffected == 1, nil
}

// incrementUnderLimit atomically increments the counter while capacity remains.
func (p *apiKeyPlugin) incrementUnderLimit(ctx context.Context, apiKey *ApiKey) error {
	res, err := p.core.UpdateWithResult(ctx, p.apiKeySchema, map[limen.SchemaField]any{
		APIKeySchemaRateLimitRequestCountField: limen.IncrementBy(1),
		APIKeySchemaLastUsedAtField:            time.Now(),
	}, []limen.Where{
		limen.Eq(p.apiKeySchema.GetIDField(), apiKey.ID),
		limen.Lt(p.apiKeySchema.GetRateLimitRequestCountField(), *apiKey.RateLimitMax),
	})
	if err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return ErrRateLimitExceeded
	}
	return nil
}
