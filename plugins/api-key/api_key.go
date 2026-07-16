package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func (p *apiKeyPlugin) generateAPIKey(profile *Profile) string {
	if profile.KeyGenerator != nil {
		return profile.KeyGenerator(profile)
	}

	key := limen.GenerateRandomString(profile.KeyLength, limen.CharSetAlphabetic)
	return profile.Prefix + key
}

func (p *apiKeyPlugin) hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// resolvePermissions returns effective key permissions, limited to what the principal can be granted.
func (p *apiKeyPlugin) resolvePermissions(ctx context.Context, profile *Profile, principalID any, customPermissions Permissions) (Permissions, error) {
	grantablePermissions, err := p.grantablePrincipalPermissions(ctx, profile.PrincipalType, principalID)
	if err != nil {
		return nil, err
	}

	permissions := profile.DefaultPermissions
	if len(customPermissions) > 0 {
		permissions = customPermissions
	}

	if len(grantablePermissions) == 0 {
		return permissions, nil
	}

	return access.ClampPermissions(permissions, grantablePermissions), nil
}

func (p *apiKeyPlugin) Create(ctx context.Context, user *limen.User, req *ApiKeyCreateRequest) (*ApiKeyCreateResult, error) {
	profile, err := p.GetProfile(req.ProfileID)
	if err != nil {
		return nil, err
	}

	principalID, err := p.resolvePrincipalID(ctx, profile.PrincipalType, user.ID)
	if err != nil {
		return nil, err
	}

	permissions, err := p.resolvePermissions(ctx, profile, principalID, req.Permissions)
	if err != nil {
		return nil, err
	}

	key := p.generateAPIKey(profile)
	payload := &ApiKey{
		Profile:         profile.ID,
		Name:            req.Name,
		CreatedByUserID: user.ID,
		Permissions:     permissions,
		ExpiresAt:       resolveExpiresAt(req.ExpiresIn),
		Prefix:          &profile.Prefix,
		Enabled:         true,
		KeyHash:         p.hashAPIKey(key),
		Last4:           key[len(key)-4:],
		PrincipalType:   profile.PrincipalType,
		PrincipalID:     principalID,
		Metadata:        req.Metadata,
	}

	if profile.HasRateLimit() {
		rateLimitWindowMS := profile.RateLimitWindow.Milliseconds()
		payload.RateLimitMax = &profile.RateLimitMax
		payload.RateLimitWindowMS = &rateLimitWindowMS
	}

	apiKey, err := p.store.CreateAndReturn(ctx, payload)
	if err != nil {
		return nil, err
	}

	return &ApiKeyCreateResult{
		Key:    key,
		ApiKey: apiKey,
	}, nil
}

func (p *apiKeyPlugin) Get(ctx context.Context, user *limen.User, id string) (*ApiKey, error) {
	apiKeyModel, err := p.core.FindOne(ctx, p.apiKeySchema, []limen.Where{
		limen.Eq(p.apiKeySchema.GetIDField(), id),
	}, nil)
	if err != nil {
		return nil, err
	}

	apiKey := apiKeyModel.(*ApiKey)
	if err := p.ensureUserOwnsAPIKey(ctx, user, apiKey); err != nil {
		return nil, err
	}
	return apiKey, nil
}

func (p *apiKeyPlugin) List(ctx context.Context, user *limen.User, profileId string, filter *ApiKeyListFilter, opts *limen.QueryOptions) (*limen.Page[*ApiKey], error) {
	profile, err := p.GetProfile(profileId)
	if err != nil {
		return nil, err
	}

	principalID, err := p.resolvePrincipalID(ctx, profile.PrincipalType, user.ID)
	if err != nil {
		return nil, err
	}

	conditions := []limen.Where{
		limen.Eq(p.apiKeySchema.GetPrincipalTypeField(), string(profile.PrincipalType)),
		limen.Eq(p.apiKeySchema.GetPrincipalIDField(), principalID),
		limen.Eq(p.apiKeySchema.GetProfileField(), profile.ID),
	}

	if filter.Status == APIKeyStatusEnabled {
		conditions = append(conditions, limen.Eq(p.apiKeySchema.GetEnabledField(), true))
	}

	if filter.Status == APIKeyStatusDisabled {
		conditions = append(conditions, limen.Eq(p.apiKeySchema.GetEnabledField(), false))
	}

	page, err := p.core.FindWithOptions(ctx, p.apiKeySchema, conditions, opts)
	if err != nil {
		return nil, err
	}

	return limen.MapPage[*ApiKey](page), nil
}

func (p *apiKeyPlugin) Update(ctx context.Context, user *limen.User, apiKeyID any, req *ApiKeyUpdateRequest) (*ApiKey, error) {
	apiKeyModel, err := p.core.FindOne(ctx, p.apiKeySchema, []limen.Where{
		limen.Eq(p.apiKeySchema.GetIDField(), apiKeyID),
	}, nil)

	if err != nil {
		return nil, err
	}

	apiKey := apiKeyModel.(*ApiKey)
	if err := p.ensureUserOwnsAPIKey(ctx, user, apiKey); err != nil {
		return nil, err
	}

	profile, err := p.GetProfile(apiKey.Profile)
	if err != nil {
		return nil, err
	}

	payload := map[limen.SchemaField]any{}

	if req.Name != "" {
		payload[APIKeySchemaNameField] = req.Name
	}

	if req.Enabled != nil && *req.Enabled != apiKey.Enabled {
		payload[APIKeySchemaEnabledField] = *req.Enabled
	}

	if req.AllPermissions || len(req.Permissions) > 0 {
		permissions, err := p.resolvePermissions(ctx, profile, apiKey.PrincipalID, req.Permissions)
		if err != nil {
			return nil, err
		}
		payload[APIKeySchemaPermissionsField] = permissions
	}

	if req.Metadata != nil {
		payload[APIKeySchemaMetadataField] = req.Metadata
	}

	updatedApiKey, err := p.store.UpdateAndReturn(ctx, apiKey, payload, []limen.Where{
		limen.Eq(p.apiKeySchema.GetIDField(), apiKeyID),
	})
	if err != nil {
		return nil, err
	}

	return updatedApiKey, nil
}

func (p *apiKeyPlugin) Revoke(ctx context.Context, user *limen.User, apiKeyId any) error {
	apiKeyModel, err := p.core.FindOne(ctx, p.apiKeySchema, []limen.Where{
		limen.Eq(p.apiKeySchema.GetIDField(), apiKeyId),
	}, nil)

	if err != nil {
		return err
	}

	apiKey := apiKeyModel.(*ApiKey)
	if err := p.ensureUserOwnsAPIKey(ctx, user, apiKey); err != nil {
		return err
	}

	return p.store.Delete(ctx, apiKey)
}

func (p *apiKeyPlugin) Rotate(ctx context.Context, user *limen.User, apiKeyId any, req *ApiKeyRotateRequest) (*ApiKeyCreateResult, error) {
	apiKeyModel, err := p.core.FindOne(ctx, p.apiKeySchema, []limen.Where{
		limen.Eq(p.apiKeySchema.GetIDField(), apiKeyId),
	}, nil)
	if err != nil {
		return nil, err
	}

	apiKey := apiKeyModel.(*ApiKey)
	if err := p.ensureUserOwnsAPIKey(ctx, user, apiKey); err != nil {
		return nil, err
	}

	profile, err := p.GetProfile(apiKey.Profile)
	if err != nil {
		return nil, err
	}

	expiresAt := resolveExpiresAt(req.ExpiresIn)
	newKey := p.generateAPIKey(profile)
	newKeyHash := p.hashAPIKey(newKey)
	payload := map[limen.SchemaField]any{
		APIKeySchemaKeyHashField:   newKeyHash,
		APIKeySchemaExpiresAtField: expiresAt,
		APIKeySchemaLast4Field:     newKey[len(newKey)-4:],
	}

	if req.AllPermissions || len(req.Permissions) > 0 {
		permissions, err := p.resolvePermissions(ctx, profile, apiKey.PrincipalID, req.Permissions)
		if err != nil {
			return nil, err
		}
		payload[APIKeySchemaPermissionsField] = permissions
	}

	rotatedApiKey, err := p.store.UpdateAndReturn(ctx, apiKey, payload, []limen.Where{
		limen.Eq(p.apiKeySchema.GetIDField(), apiKeyId),
	})
	if err != nil {
		return nil, err
	}

	return &ApiKeyCreateResult{
		Key:    newKey,
		ApiKey: rotatedApiKey,
	}, nil
}

func (p *apiKeyPlugin) ensureUserOwnsAPIKey(ctx context.Context, user *limen.User, apiKey *ApiKey) error {
	principalID, err := p.resolvePrincipalID(ctx, apiKey.PrincipalType, user.ID)
	if err != nil {
		return err
	}
	if principalID != apiKey.PrincipalID {
		return limen.ErrForbidden
	}
	return nil
}

func resolveExpiresAt(expiresIn *int64) *time.Time {
	if expiresIn != nil {
		expiresAt := time.Now().Add(time.Duration(*expiresIn) * time.Second)
		return &expiresAt
	}
	return nil
}
