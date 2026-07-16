package apikey

import (
	"context"
	"errors"
	"time"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func (p *apiKeyPlugin) Verify(ctx context.Context, key string, requiredPermissions access.Permissions) (*ApiKey, error) {
	return p.ValidateApiKey(ctx, key, requiredPermissions, "", p.config.rateLimitStoreType != limen.StoreTypeCache)
}

func (p *apiKeyPlugin) VerifyWithProfile(ctx context.Context, key string, requiredPermissions access.Permissions, profileID string) (*ApiKey, error) {
	return p.ValidateApiKey(ctx, key, requiredPermissions, profileID, p.config.rateLimitStoreType != limen.StoreTypeCache)
}

func (p *apiKeyPlugin) ValidateApiKey(ctx context.Context, key string, requiredPermissions access.Permissions, profileID string, skipCache bool) (*ApiKey, error) {
	keyHash := p.hashAPIKey(key)

	apiKey, err := p.store.FindOne(ctx, keyHash, skipCache)

	if err != nil {
		if errors.Is(err, limen.ErrRecordNotFound) {
			return nil, ErrInvalidAPIKey
		}
		return nil, err
	}

	profile, err := p.resolveVerificationProfile(apiKey, profileID)
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

	if err := p.rateLimiter.Enforce(ctx, apiKey); err != nil {
		return nil, err
	}

	if len(requiredPermissions) > 0 {
		if !access.HasRequiredPermissions(apiKey.Permissions, requiredPermissions) {
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
