package apikey

import (
	"fmt"
	"time"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type Permissions = access.Permissions

type ApiKeyCreateRequest struct {
	ProfileID   string         `json:"profile"`
	Name        string         `json:"name"`
	Permissions Permissions    `json:"permissions,omitempty"`
	ExpiresIn   *int64         `json:"expires_in,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type ApiKeyUpdateRequest struct {
	Name           string         `json:"name"`
	Permissions    Permissions    `json:"permissions,omitempty"`
	AllPermissions bool           `json:"all_permissions,omitempty"`
	Enabled        *bool          `json:"enabled,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type ApiKeyRotateRequest struct {
	ExpiresIn      *int64      `json:"expires_in,omitempty"`
	Permissions    Permissions `json:"permissions,omitempty"`
	AllPermissions bool        `json:"all_permissions,omitempty"`
}

type ApiKeyCreateResult struct {
	Key string `json:"key"`
	*ApiKey
}

type ApiKeyListFilter struct {
	// Return only API keys for the given profile.
	ProfileID string `json:"profile"`
	// Return only enabled API keys.
	Status ApiKeyStatus `json:"status,omitempty"`
}

type config struct {
	profiles           map[string]Profile
	rateLimitStoreType limen.StoreType
	cacheEnabled       bool
	cacheTTL           time.Duration
	lastUsedAtThrottle time.Duration
	metadataFilter     func(metadata map[string]any) map[string]any
}

type ConfigOption func(*config)

func WithProfiles(profiles ...Profile) ConfigOption {
	return func(c *config) {
		for index, profile := range profiles {
			if _, ok := c.profiles[profile.ID]; ok && profile.ID != "default" {
				panic(fmt.Sprintf("api-key: profile %q already exists", profile.ID))
			}

			profile.applyDefaults()

			if err := profile.validate(); err != nil {
				panic(fmt.Sprintf("api-key: profile at index %d is invalid: %s", index, err))
			}

			c.profiles[profile.ID] = profile
		}
	}
}

// WithRateLimitStoreType sets the type of rate limit store to use.
// This allows the rate limiting to be performed against a different store than the one used for persistence.
// like redis, memory, or primary database.
func WithRateLimitStoreType(storeType limen.StoreType) ConfigOption {
	return func(c *config) {
		c.rateLimitStoreType = storeType
	}
}

// WithDisableCache disables caching of API keys.
// If disabled, the API key will be fetched from the database on every find operation.
func WithDisableCache() ConfigOption {
	return func(c *config) {
		c.cacheEnabled = false
	}
}

// WithCacheTTL sets the TTL for the cache store.
// Default is 5 minutes.
func WithCacheTTL(ttl time.Duration) ConfigOption {
	return func(c *config) {
		c.cacheTTL = ttl
	}
}

// WithLastUsedAtThrottle limits how often LastUsedAt is persisted when using
// cache-backed rate limiting. Database-backed rate limiting updates LastUsedAt
// as part of its normal rate-limit operations and does not use this setting.
func WithLastUsedAtThrottle(throttle time.Duration) ConfigOption {
	return func(c *config) {
		c.lastUsedAtThrottle = throttle
	}
}

// WithMetadataFilter sets a function that maps an API key's stored metadata to the
// metadata returned in HTTP responses (e.g. to allowlist or redact keys). Without
// it, metadata is not returned to clients.
func WithMetadataFilter(filter func(metadata map[string]any) map[string]any) ConfigOption {
	return func(c *config) {
		c.metadataFilter = filter
	}
}

// filterMetadata applies the configured metadata filter for HTTP responses.
func (c *config) filterMetadata(metadata map[string]any) map[string]any {
	if c.metadataFilter == nil {
		return nil
	}
	return c.metadataFilter(metadata)
}
