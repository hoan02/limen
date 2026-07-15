package apikey

import (
	"context"

	"github.com/thecodearcher/limen"
)

// API is the public interface for the api-key plugin.
// Call the Use() function to obtain a type-safe reference from a Limen instance.
type API interface {
	// Create issues a new API key. The plaintext secret is returned only once.
	Create(ctx context.Context, user *limen.User, req *ApiKeyCreateRequest) (*ApiKeyCreateResult, error)

	// List returns API keys created by the user. If filter is provided, only API keys matching the filter will be returned.
	List(ctx context.Context, user *limen.User, filter *ApiKeyListFilter, opts *limen.QueryOptions) (*limen.Page[*ApiKey], error)

	// Update changes an existing API key.
	Update(ctx context.Context, user *limen.User, apiKeyID any, req *ApiKeyUpdateRequest) (*ApiKey, error)

	// Verify authenticates an API key and returns its ApiKey model.
	Verify(ctx context.Context, key string, opts *VerifyOptions) (*ApiKey, error)

	// Revoke disables or deletes an API key.
	Revoke(ctx context.Context, user *limen.User, apiKeyId any, isTemporary bool) error

	// Rotate replaces an API key's secret. The new plaintext secret is returned only once.
	Rotate(ctx context.Context, user *limen.User, apiKeyId any, req *ApiKeyRotateRequest) (*ApiKeyCreateResult, error)
}

// Use returns a type-safe API for the api-key plugin.
// Panics if the plugin was not registered in Config.Plugins,
// making it suitable for method chaining.
func Use(a *limen.Limen) API {
	return limen.Use[API](a, limen.PluginAPIKey)
}

var _ API = (*apiKeyPlugin)(nil)
