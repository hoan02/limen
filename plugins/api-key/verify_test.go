package apikey

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func TestAPIKeyPlugin_Verify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		key                 string
		permissions         access.Permissions
		profiles            []Profile
		prepare             func(t *testing.T, plugin *apiKeyPlugin, user *limen.User) string
		expectedError       error
		expectLastUsedAtSet bool
	}{
		{
			name: "valid key",
			prepare: func(t *testing.T, plugin *apiKeyPlugin, user *limen.User) string {
				return createTestAPIKey(t, plugin, user, "default", nil, nil)
			},
			expectLastUsedAtSet: true,
		},
		{
			name: "valid key with custom profile",
			profiles: []Profile{
				{
					ID:            "service",
					PrincipalType: PrincipalTypeUser,
					Prefix:        "service_",
				},
			},
			prepare: func(t *testing.T, plugin *apiKeyPlugin, user *limen.User) string {
				return createTestAPIKey(t, plugin, user, "service", nil, nil)
			},
			expectLastUsedAtSet: true,
		},
		{
			name:          "unknown key",
			key:           "sk_unknown",
			expectedError: ErrInvalidAPIKey,
		},
		{
			name: "disabled key",
			prepare: func(t *testing.T, plugin *apiKeyPlugin, user *limen.User) string {
				key := createTestAPIKey(t, plugin, user, "default", nil, nil)
				apiKey := findTestAPIKey(t, plugin, key)
				require.NoError(t, plugin.core.Update(
					t.Context(),
					plugin.apiKeySchema,
					map[limen.SchemaField]any{APIKeySchemaEnabledField: false},
					[]limen.Where{limen.Eq(plugin.apiKeySchema.GetIDField(), apiKey.ID)},
				))
				return key
			},
			expectedError: ErrAPIKeyDisabled,
		},
		{
			name: "expired key",
			prepare: func(t *testing.T, plugin *apiKeyPlugin, user *limen.User) string {
				expiresIn := int64(-1)
				return createTestAPIKey(t, plugin, user, "default", nil, &expiresIn)
			},
			expectedError: ErrAPIKeyExpired,
		},
		{
			name:        "allowed permission",
			permissions: access.Permissions{"file": {"read"}},
			prepare: func(t *testing.T, plugin *apiKeyPlugin, user *limen.User) string {
				return createTestAPIKey(
					t,
					plugin,
					user,
					"default",
					access.Permissions{"file": {"read"}},
					nil,
				)
			},
			expectLastUsedAtSet: true,
		},
		{
			name:        "missing permission",
			permissions: access.Permissions{"file": {"write"}},
			prepare: func(t *testing.T, plugin *apiKeyPlugin, user *limen.User) string {
				return createTestAPIKey(
					t,
					plugin,
					user,
					"default",
					access.Permissions{"file": {"read"}},
					nil,
				)
			},
			expectedError: ErrInsufficientPermissions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAPIKeyPlugin(t, tt.profiles...)
			user := limen.SeedTestUser(t, l, tt.name+"@test.com")

			key := tt.key
			if tt.prepare != nil {
				key = tt.prepare(t, plugin, user)
			}

			got, err := plugin.Verify(t.Context(), key, tt.permissions)
			if tt.expectedError != nil {
				require.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			if tt.expectLastUsedAtSet {
				assert.NotNil(t, findTestAPIKey(t, plugin, key).LastUsedAt)
			}
		})
	}
}

func TestAPIKeyPlugin_VerifyWithProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		profileID     string
		expectedError error
	}{
		{
			name:      "matching profile",
			profileID: "service",
		},
		{
			name:          "mismatched profile",
			profileID:     "default",
			expectedError: ErrInvalidAPIKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAPIKeyPlugin(t, Profile{
				ID:            "service",
				PrincipalType: PrincipalTypeUser,
				Prefix:        "service_",
			})
			user := limen.SeedTestUser(t, l, tt.name+"@test.com")
			key := createTestAPIKey(t, plugin, user, "service", nil, nil)

			got, err := plugin.VerifyWithProfile(t.Context(), key, nil, tt.profileID)
			if tt.expectedError != nil {
				require.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "service", got.Profile)
		})
	}
}

func TestAPIKeyPlugin_Verify_EnforcesLimitConcurrently(t *testing.T) {
	t.Parallel()

	const (
		maxRequests   = 5
		totalRequests = 20
	)

	idleTimeout := time.Minute
	l, plugin := newTestAPIKeyPlugin(t, Profile{
		ID:              "limited",
		PrincipalType:   PrincipalTypeUser,
		Prefix:          "limited_",
		RateLimitMax:    maxRequests,
		RateLimitWindow: &idleTimeout,
	})
	user := limen.SeedTestUser(t, l, "concurrent-limit@test.com")
	key := createTestAPIKey(t, plugin, user, "limited", nil, nil)

	succeeded, rejected := verifyConcurrently(t, plugin, key, totalRequests)

	assert.Equal(t, maxRequests, succeeded)
	assert.Equal(t, totalRequests-maxRequests, rejected)
	apiKey := findTestAPIKey(t, plugin, key)
	require.NotNil(t, apiKey.RateLimitRequestCount)
	assert.Equal(t, int32(maxRequests), *apiKey.RateLimitRequestCount)
}

func TestAPIKeyPlugin_Verify_ResetsAfterIdleTimeoutConcurrently(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		const (
			maxRequests   = 3
			totalRequests = 10
		)

		idleTimeout := 1 * time.Second
		l, plugin := newTestAPIKeyPlugin(t, Profile{
			ID:              "idle-limited",
			PrincipalType:   PrincipalTypeUser,
			Prefix:          "idle_",
			RateLimitMax:    maxRequests,
			RateLimitWindow: &idleTimeout,
		})
		user := limen.SeedTestUser(t, l, "idle-reset@test.com")
		key := createTestAPIKey(t, plugin, user, "idle-limited", nil, nil)

		succeeded, rejected := verifyConcurrently(t, plugin, key, totalRequests)
		require.Equal(t, maxRequests, succeeded)
		require.Equal(t, totalRequests-maxRequests, rejected)

		time.Sleep(idleTimeout)

		succeeded, rejected = verifyConcurrently(t, plugin, key, totalRequests)
		assert.Equal(t, maxRequests, succeeded)
		assert.Equal(t, totalRequests-maxRequests, rejected)
		apiKey := findTestAPIKey(t, plugin, key)
		require.NotNil(t, apiKey.RateLimitRequestCount)
		assert.Equal(t, int32(maxRequests), *apiKey.RateLimitRequestCount)
	})
}

func newTestAPIKeyPlugin(t *testing.T, profiles ...Profile) (*limen.Limen, *apiKeyPlugin) {
	t.Helper()

	options := []ConfigOption{}
	if len(profiles) > 0 {
		options = append(options, WithProfiles(profiles...))
	}

	plugin := New(options...)
	plugin.config.rateLimitStoreType = limen.StoreTypeDatabase
	l, _ := limen.NewTestLimen(t, plugin)
	return l, plugin
}

func createTestAPIKey(
	t *testing.T,
	plugin *apiKeyPlugin,
	user *limen.User,
	profileID string,
	permissions Permissions,
	expiresIn *int64,
) string {
	t.Helper()

	result, err := plugin.Create(t.Context(), user, &ApiKeyCreateRequest{
		ProfileID:   profileID,
		Name:        "test key",
		Permissions: permissions,
		ExpiresIn:   expiresIn,
	})
	require.NoError(t, err)
	return result.Key
}

func findTestAPIKey(t *testing.T, plugin *apiKeyPlugin, key string) *ApiKey {
	t.Helper()

	model, err := plugin.core.FindOne(t.Context(), plugin.apiKeySchema, []limen.Where{
		limen.Eq(plugin.apiKeySchema.GetKeyHashField(), plugin.hashAPIKey(key)),
	}, nil)
	require.NoError(t, err)
	return model.(*ApiKey)
}

func verifyConcurrently(t *testing.T, plugin *apiKeyPlugin, key string, requests int) (succeeded, rejected int) {
	t.Helper()

	start := make(chan struct{})
	results := make(chan error, requests)

	var group sync.WaitGroup
	for range requests {
		group.Go(func() {
			<-start
			_, err := plugin.Verify(context.Background(), key, access.Permissions{})
			results <- err
		})
	}

	close(start)
	group.Wait()
	close(results)

	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrRateLimitExceeded):
			rejected++
		default:
			require.NoError(t, err)
		}
	}

	return succeeded, rejected
}
