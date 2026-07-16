package apikey

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thecodearcher/limen"
)

type apiKeyStore struct {
	self         *apiKeyPlugin
	apiKeySchema *apiKeySchema
	core         *limen.LimenCore
	cache        limen.CacheAdapter
	prefix       string
}

func newApiKeyStore(p *apiKeyPlugin) *apiKeyStore {
	return &apiKeyStore{
		self:         p,
		apiKeySchema: p.apiKeySchema,
		core:         p.core,
		cache:        p.core.CacheStore(),
		prefix:       p.core.CacheKeyPrefix(),
	}
}

func (s *apiKeyStore) storeKey(key string) string {
	return fmt.Sprintf("%s:api-key:%s", s.prefix, key)
}

func (s *apiKeyStore) getFromCache(ctx context.Context, keyHash string) (*ApiKey, error) {
	data, err := s.cache.Get(ctx, s.storeKey(keyHash))
	if err != nil && err != limen.ErrRecordNotFound {
		return nil, err
	}

	if data == nil {
		return nil, nil
	}

	var apiKey ApiKey
	if err := json.Unmarshal(data, &apiKey); err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (s *apiKeyStore) updateCache(ctx context.Context, apiKey *ApiKey) error {
	if !s.self.config.cacheEnabled {
		return nil
	}

	data, err := json.Marshal(apiKey)
	if err != nil {
		return fmt.Errorf("failed to marshal api key: %w", err)
	}

	return s.cache.Set(ctx, s.storeKey(apiKey.KeyHash), data, s.self.config.cacheTTL)
}

func (s *apiKeyStore) invalidateCache(ctx context.Context, keyHash string) error {
	if !s.self.config.cacheEnabled {
		return nil
	}

	return s.cache.Delete(ctx, s.storeKey(keyHash))
}

func (s *apiKeyStore) FindOne(ctx context.Context, keyHash string, skipCache bool) (*ApiKey, error) {
	if !skipCache && s.self.config.cacheEnabled {
		apiKey, err := s.getFromCache(ctx, keyHash)
		if err != nil {
			return nil, err
		}

		if apiKey != nil {
			return apiKey, nil
		}
	}

	apiKeyModel, err := s.core.FindOne(ctx, s.apiKeySchema, []limen.Where{
		limen.Eq(s.apiKeySchema.GetKeyHashField(), keyHash),
	}, nil)
	if err != nil {
		return nil, err
	}

	if !skipCache {
		s.updateCache(ctx, apiKeyModel.(*ApiKey))
	}

	return apiKeyModel.(*ApiKey), nil
}

func (s *apiKeyStore) Update(ctx context.Context, apiKey *ApiKey, value map[limen.SchemaField]any, conditions []limen.Where) error {
	s.invalidateCache(ctx, apiKey.KeyHash)
	return s.core.Update(ctx, s.apiKeySchema, value, conditions)
}

func (s *apiKeyStore) UpdateAndReturn(ctx context.Context, apiKey *ApiKey, value map[limen.SchemaField]any, conditions []limen.Where) (*ApiKey, error) {
	s.invalidateCache(ctx, apiKey.KeyHash)

	apiKeyModel, err := s.core.UpdateAndReturn(ctx, s.apiKeySchema, value, conditions, apiKey.ID)
	if err != nil {
		return nil, err
	}

	return apiKeyModel.(*ApiKey), nil
}

func (s *apiKeyStore) CreateAndReturn(ctx context.Context, value *ApiKey) (*ApiKey, error) {
	apiKeyModel, err := s.core.CreateAndReturn(ctx, s.apiKeySchema, value, nil, APIKeySchemaKeyHashField)
	if err != nil {
		return nil, err
	}

	apiKey := apiKeyModel.(*ApiKey)
	s.updateCache(ctx, apiKey)

	return apiKey, nil
}

func (s *apiKeyStore) Delete(ctx context.Context, apiKey *ApiKey) error {
	s.invalidateCache(ctx, apiKey.KeyHash)
	return s.core.Delete(ctx, s.apiKeySchema, []limen.Where{
		limen.Eq(s.apiKeySchema.GetIDField(), apiKey.ID),
	})
}
