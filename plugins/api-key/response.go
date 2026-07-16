package apikey

import (
	"time"

	"github.com/thecodearcher/limen"
)

// ApiKeyResponse is the public HTTP representation of an API key.
type ApiKeyResponse struct {
	ID          any                 `json:"id"`
	Name        string              `json:"name"`
	Profile     string              `json:"profile"`
	Prefix      *string             `json:"prefix"`
	Last4       string              `json:"last4"`
	Permissions map[string][]string `json:"permissions"`
	Enabled     bool                `json:"enabled"`
	ExpiresAt   *time.Time          `json:"expires_at"`
	IsExpired   bool                `json:"is_expired"`
	LastUsedAt  *time.Time          `json:"last_used_at"`
	Metadata    map[string]any      `json:"metadata"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// ApiKeyCreateResponse includes the plaintext key returned by create and rotate operations.
type ApiKeyCreateResponse struct {
	Key string `json:"key"`
	*ApiKeyResponse
}

func newApiKeyResponse(apiKey *ApiKey, cfg *config) *ApiKeyResponse {
	return &ApiKeyResponse{
		ID:          apiKey.ID,
		Name:        apiKey.Name,
		Profile:     apiKey.Profile,
		Prefix:      apiKey.Prefix,
		Last4:       apiKey.Last4,
		Permissions: apiKey.Permissions,
		Enabled:     apiKey.Enabled,
		ExpiresAt:   apiKey.ExpiresAt,
		IsExpired:   apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()),
		LastUsedAt:  apiKey.LastUsedAt,
		Metadata:    cfg.filterMetadata(apiKey.Metadata),
		CreatedAt:   apiKey.CreatedAt,
		UpdatedAt:   apiKey.UpdatedAt,
	}
}

func newApiKeyCreateResponse(result *ApiKeyCreateResult, cfg *config) *ApiKeyCreateResponse {
	return &ApiKeyCreateResponse{
		Key:            result.Key,
		ApiKeyResponse: newApiKeyResponse(result.ApiKey, cfg),
	}
}

func newApiKeyPageResponse(page *limen.Page[*ApiKey], cfg *config) *limen.Page[*ApiKeyResponse] {
	items := make([]*ApiKeyResponse, 0, len(page.Items))
	for _, apiKey := range page.Items {
		items = append(items, newApiKeyResponse(apiKey, cfg))
	}

	return &limen.Page[*ApiKeyResponse]{
		Items:      items,
		Total:      page.Total,
		Page:       page.Page,
		PerPage:    page.PerPage,
		TotalPages: page.TotalPages,
	}
}
