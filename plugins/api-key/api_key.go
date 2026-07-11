package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"slices"

	"github.com/thecodearcher/limen"
)

type ApiKeyCreateRequest struct {
	ProfileID   string      `json:"profile"`
	Name        string      `json:"name"`
	Permissions Permissions `json:"permissions,omitempty"`
	ExpiresIn   *int64      `json:"expires_in,omitempty"`
}

type ApiKeyUpdateRequest struct {
	Name           string      `json:"name"`
	Permissions    Permissions `json:"permissions,omitempty"`
	AllPermissions bool        `json:"all_permissions,omitempty"`
}

type ApiKeyCreateResult struct {
	Key string `json:"key"`
	*ApiKey
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

	permissions, err := p.resolvePermissions(ctx, profile, user, req.Permissions)
	if err != nil {
		return nil, err
	}

	key := p.generateAPIKey(profile)
	payload := &ApiKey{
		Profile:         profile.ID,
		Name:            req.Name,
		CreatedByUserID: user.ID,
		Permissions:     permissions,
		ExpiresAt:       profile.ExpiresAt(req.ExpiresIn),
		Prefix:          &profile.Prefix,
		Enabled:         true,
		KeyHash:         p.hashAPIKey(key),
		Last4:           key[len(key)-4:],
		PrincipalType:   profile.PrincipalType,
		PrincipalID:     principalID,
	}

	if profile.HasRateLimit() {
		rateLimitWindowMS := profile.RateLimitWindow.Milliseconds()
		payload.RateLimitMax = &profile.RateLimitMax
		payload.RateLimitWindowMS = &rateLimitWindowMS
	}

	apiKey, err := p.core.CreateAndReturn(ctx, p.apiKeySchema, payload, nil, APIKeySchemaKeyHashField)
	if err != nil {
		return nil, err
	}

	return &ApiKeyCreateResult{
		Key:    key,
		ApiKey: apiKey.(*ApiKey),
	}, nil
}

func (p *apiKeyPlugin) generateAPIKey(profile *Profile) string {
	if p.config.generateKey != nil {
		return p.config.generateKey(profile)
	}

	key := limen.GenerateRandomString(p.config.keyLength, limen.CharSetAlphabetic)
	return profile.Prefix + key
}

func (p *apiKeyPlugin) hashAPIKey(key string) string {
	if p.config.hashKey != nil {
		return p.config.hashKey(key)
	}

	sum := sha256.Sum256([]byte(key))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (p *apiKeyPlugin) List(ctx context.Context, user *limen.User, profileID string, enabledOnly bool, opts *limen.QueryOptions) (*limen.Page[*ApiKey], error) {
	var profile *Profile

	if profileID != "" {
		selectedProfile, err := p.GetProfile(profileID)
		if err != nil {
			return nil, err
		}
		profile = selectedProfile
	}

	conditions := []limen.Where{limen.Eq(p.apiKeySchema.GetCreatedByField(), user.ID)}
	if profile != nil {
		conditions = append(conditions, limen.Eq(p.apiKeySchema.GetProfileField(), profile.ID))
	}

	if enabledOnly {
		conditions = append(conditions, limen.Eq(p.apiKeySchema.GetEnabledField(), true))
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

	profile, err := p.GetProfile(apiKey.Profile)
	if err != nil {
		return nil, err
	}

	payload := map[limen.SchemaField]any{}

	if req.Name != "" {
		payload[APIKeySchemaNameField] = req.Name
	}

	if req.AllPermissions {
		payload[APIKeySchemaPermissionsField] = profile.DefaultPermissions
	}

	if len(req.Permissions) > 0 {
		permissions, err := p.resolvePermissions(ctx, profile, user, req.Permissions)
		if err != nil {
			return nil, err
		}
		payload[APIKeySchemaPermissionsField] = permissions
	}

	updatedApiKey, err := p.core.UpdateAndReturn(ctx, p.apiKeySchema, payload, []limen.Where{
		limen.Eq(p.apiKeySchema.GetIDField(), apiKeyID),
	}, apiKeyID)
	if err != nil {
		return nil, err
	}

	return updatedApiKey.(*ApiKey), nil
}

// resolvePermissions returns effective key permissions, limited to what the principal can be granted.
func (p *apiKeyPlugin) resolvePermissions(ctx context.Context, profile *Profile, user *limen.User, customPermissions Permissions) (Permissions, error) {
	grantablePermissions, err := p.grantablePrincipalPermissions(ctx, profile.PrincipalType, user.ID)
	if err != nil {
		return nil, err
	}

	if len(customPermissions) > 0 {
		return clampPermissions(customPermissions, grantablePermissions), nil
	}

	return clampPermissions(profile.DefaultPermissions, grantablePermissions), nil
}

func (p *apiKeyPlugin) Revoke(ctx context.Context, user *limen.User, apiKeyId any, isTemporary bool) error {
	apiKeyModel, err := p.core.FindOne(ctx, p.apiKeySchema, []limen.Where{
		limen.Eq(p.apiKeySchema.GetIDField(), apiKeyId),
	}, nil)

	if err != nil {
		return err
	}

	apiKey := apiKeyModel.(*ApiKey)
	principalID, err := p.resolvePrincipalID(ctx, apiKey.PrincipalType, user.ID)
	if err != nil {
		return err
	}

	if principalID != apiKey.PrincipalID {
		return limen.ErrForbidden
	}

	if isTemporary {
		return p.core.Update(ctx, p.apiKeySchema, map[limen.SchemaField]any{
			APIKeySchemaEnabledField: false,
		}, []limen.Where{
			limen.Eq(p.apiKeySchema.GetIDField(), apiKeyId),
		})
	}

	return p.core.Delete(ctx, p.apiKeySchema, []limen.Where{
		limen.Eq(p.apiKeySchema.GetIDField(), apiKeyId),
	})
}

func intersectPerms(selected, allowed []string) []string {
	var out []string
	for _, perm := range selected {
		if slices.Contains(allowed, perm) {
			out = append(out, perm)
		}
	}
	return out
}

func clampPermissions(selected, grantable Permissions) Permissions {
	if len(grantable) == 0 {
		return selected
	}

	out := make(Permissions, len(selected))
	for resource, perms := range selected {
		if filtered := intersectPerms(perms, grantable[resource]); len(filtered) > 0 {
			out[resource] = filtered
		}
	}
	return out
}
