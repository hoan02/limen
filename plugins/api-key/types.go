package apikey

import (
	"log"

	"github.com/thecodearcher/limen/access"
)

type Permissions = access.Permissions

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
	Enabled        *bool       `json:"enabled,omitempty"`
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
	profiles map[string]Profile
}

type ConfigOption func(*config)

func WithProfiles(profiles ...Profile) ConfigOption {
	return func(c *config) {
		for _, profile := range profiles {
			if _, ok := c.profiles[profile.ID]; ok {
				log.Fatalf("profile %s already exists", profile.ID)
			}

			if profile.ID == "" {
				log.Fatalf("profile ID is required")
			}

			c.profiles[profile.ID] = profile
		}
	}
}
