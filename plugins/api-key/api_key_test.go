package apikey

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestAPIKeyPlugin_Create(t *testing.T) {
	t.Parallel()

	const generatedKey = "service_secret_1234"
	rateLimitWindow := 2 * time.Minute
	profile := Profile{
		ID:                 "service",
		PrincipalType:      PrincipalTypeUser,
		Prefix:             "service_",
		DefaultPermissions: Permissions{"file": {"read"}},
		RateLimitMax:       10,
		RateLimitWindow:    &rateLimitWindow,
		KeyGenerator: func(*Profile) string {
			return generatedKey
		},
	}
	l, plugin := newTestAPIKeyPlugin(t, profile)
	user := limen.SeedTestUser(t, l, "create@test.com")
	expiresIn := int64(60)
	beforeCreate := time.Now()

	result, err := plugin.Create(t.Context(), user, &ApiKeyCreateRequest{
		ProfileID: "service",
		Name:      "service key",
		ExpiresIn: &expiresIn,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, generatedKey, result.Key)
	assert.Equal(t, "service key", result.Name)
	assert.Equal(t, "service", result.Profile)
	assert.Equal(t, user.ID, result.CreatedByUserID)
	assert.Equal(t, PrincipalTypeUser, result.PrincipalType)
	assert.Equal(t, user.ID, result.PrincipalID)
	assert.Equal(t, "service_", *result.Prefix)
	assert.Equal(t, "1234", result.Last4)
	assert.True(t, result.Enabled)
	assert.Equal(t, map[string][]string{"file": {"read"}}, result.Permissions)
	assert.Equal(t, plugin.hashAPIKey(generatedKey), result.KeyHash)
	assert.NotEqual(t, generatedKey, result.KeyHash)
	require.NotNil(t, result.ExpiresAt)
	assert.WithinRange(t, *result.ExpiresAt, beforeCreate.Add(time.Minute), time.Now().Add(time.Minute))
	require.NotNil(t, result.RateLimitMax)
	assert.Equal(t, int32(10), *result.RateLimitMax)
	require.NotNil(t, result.RateLimitWindowMS)
	assert.Equal(t, rateLimitWindow.Milliseconds(), *result.RateLimitWindowMS)

	persisted := findTestAPIKey(t, plugin, generatedKey)
	assert.Equal(t, result.ID, persisted.ID)
	assert.Equal(t, result.KeyHash, persisted.KeyHash)
}

func TestAPIKeyPlugin_Create_ClampsPermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		requested   Permissions
		expected    map[string][]string
		defaultPerm Permissions
	}{
		{
			name:        "requested permissions",
			requested:   Permissions{"file": {"read", "delete"}, "admin": {"write"}},
			expected:    map[string][]string{"file": {"read"}},
			defaultPerm: Permissions{"file": {"write"}},
		},
		{
			name:        "profile defaults",
			expected:    map[string][]string{"file": {"read"}},
			defaultPerm: Permissions{"file": {"read", "delete"}, "admin": {"write"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAPIKeyPlugin(t, Profile{
				ID:                 "restricted",
				PrincipalType:      PrincipalTypeUser,
				Prefix:             "restricted_",
				DefaultPermissions: tt.defaultPerm,
			})
			plugin.RegisterPrincipalResolver(string(PrincipalTypeUser), &testPrincipalResolver{
				grantable: Permissions{"file": {"read", "write"}},
			})
			user := limen.SeedTestUser(t, l, tt.name+"@test.com")

			result, err := plugin.Create(t.Context(), user, &ApiKeyCreateRequest{
				ProfileID:   "restricted",
				Name:        "restricted key",
				Permissions: tt.requested,
			})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Permissions)
		})
	}
}

func TestAPIKeyPlugin_Create_RejectsUnknownProfile(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAPIKeyPlugin(t)
	user := limen.SeedTestUser(t, l, "unknown-profile@test.com")

	result, err := plugin.Create(t.Context(), user, &ApiKeyCreateRequest{
		ProfileID: "unknown",
		Name:      "unknown profile key",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, http.StatusNotFound, limen.ToLimenError(err).Status())
}

func TestAPIKeyPlugin_List(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAPIKeyPlugin(t, Profile{
		ID:            "service",
		PrincipalType: PrincipalTypeUser,
		Prefix:        "service_",
	})
	user := limen.SeedTestUser(t, l, "list@test.com")
	otherUser := limen.SeedTestUser(t, l, "list-other@test.com")

	enabledKey := createTestAPIKey(t, plugin, user, "default", nil, nil)
	disabledKey := createTestAPIKey(t, plugin, user, "default", nil, nil)
	createTestAPIKey(t, plugin, user, "service", nil, nil)
	createTestAPIKey(t, plugin, otherUser, "default", nil, nil)

	disabledAPIKey := findTestAPIKey(t, plugin, disabledKey)
	require.NoError(t, plugin.core.Update(
		t.Context(),
		plugin.apiKeySchema,
		map[limen.SchemaField]any{APIKeySchemaEnabledField: false},
		[]limen.Where{limen.Eq(plugin.apiKeySchema.GetIDField(), disabledAPIKey.ID)},
	))

	tests := []struct {
		name        string
		filter      *ApiKeyListFilter
		options     *limen.QueryOptions
		expectedIDs []any
		expectedAll int64
	}{
		{
			name:        "profile and owner",
			filter:      &ApiKeyListFilter{ProfileID: "default"},
			expectedIDs: []any{findTestAPIKey(t, plugin, enabledKey).ID, disabledAPIKey.ID},
			expectedAll: 2,
		},
		{
			name:        "enabled",
			filter:      &ApiKeyListFilter{ProfileID: "default", Status: APIKeyStatusEnabled},
			expectedIDs: []any{findTestAPIKey(t, plugin, enabledKey).ID},
			expectedAll: 1,
		},
		{
			name:        "disabled",
			filter:      &ApiKeyListFilter{ProfileID: "default", Status: APIKeyStatusDisabled},
			expectedIDs: []any{disabledAPIKey.ID},
			expectedAll: 1,
		},
		{
			name:        "pagination",
			filter:      &ApiKeyListFilter{ProfileID: "default"},
			options:     &limen.QueryOptions{Limit: 1, Offset: 1},
			expectedAll: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := plugin.List(t.Context(), user, "default", tt.filter, tt.options)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedAll, page.Total)
			if tt.options != nil {
				assert.Len(t, page.Items, tt.options.Limit)
				return
			}

			actualIDs := make([]any, 0, len(page.Items))
			for _, apiKey := range page.Items {
				actualIDs = append(actualIDs, apiKey.ID)
			}
			assert.ElementsMatch(t, tt.expectedIDs, actualIDs)
		})
	}
}

func TestAPIKeyPlugin_Update(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAPIKeyPlugin(t)
	user := limen.SeedTestUser(t, l, "update@test.com")
	key := createTestAPIKey(
		t,
		plugin,
		user,
		"default",
		Permissions{"file": {"read"}},
		nil,
	)
	apiKey := findTestAPIKey(t, plugin, key)
	enabled := false

	updated, err := plugin.Update(t.Context(), user, apiKey.ID, &ApiKeyUpdateRequest{
		Name:        "updated key",
		Permissions: Permissions{"file": {"write"}},
		Enabled:     &enabled,
	})

	require.NoError(t, err)
	assert.Equal(t, apiKey.ID, updated.ID)
	assert.Equal(t, "updated key", updated.Name)
	assert.False(t, updated.Enabled)
	assert.Equal(t, map[string][]string{"file": {"write"}}, updated.Permissions)
	assert.Equal(t, apiKey.KeyHash, updated.KeyHash)
	assert.Equal(t, apiKey.Profile, updated.Profile)
}

func TestAPIKeyPlugin_Update_EnforcesOwnership(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAPIKeyPlugin(t)
	owner := limen.SeedTestUser(t, l, "update-owner@test.com")
	otherUser := limen.SeedTestUser(t, l, "update-other@test.com")
	key := createTestAPIKey(t, plugin, owner, "default", nil, nil)
	apiKey := findTestAPIKey(t, plugin, key)

	updated, err := plugin.Update(t.Context(), otherUser, apiKey.ID, &ApiKeyUpdateRequest{
		Name: "not allowed",
	})

	require.ErrorIs(t, err, limen.ErrForbidden)
	assert.Nil(t, updated)
	assert.Equal(t, "test key", findTestAPIKey(t, plugin, key).Name)
}

func TestAPIKeyPlugin_Update_ReturnsNotFound(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAPIKeyPlugin(t)
	user := limen.SeedTestUser(t, l, "update-missing@test.com")

	updated, err := plugin.Update(t.Context(), user, "missing", &ApiKeyUpdateRequest{
		Name: "missing",
	})

	require.ErrorIs(t, err, limen.ErrRecordNotFound)
	assert.Nil(t, updated)
}

func TestAPIKeyPlugin_Revoke(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAPIKeyPlugin(t)
	user := limen.SeedTestUser(t, l, "revoke@test.com")
	key := createTestAPIKey(t, plugin, user, "default", nil, nil)
	apiKey := findTestAPIKey(t, plugin, key)

	require.NoError(t, plugin.Revoke(t.Context(), user, apiKey.ID))

	verified, err := plugin.Verify(t.Context(), key, nil)
	require.ErrorIs(t, err, ErrInvalidAPIKey)
	assert.Nil(t, verified)
}

func TestAPIKeyPlugin_Revoke_EnforcesOwnership(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAPIKeyPlugin(t)
	owner := limen.SeedTestUser(t, l, "revoke-owner@test.com")
	otherUser := limen.SeedTestUser(t, l, "revoke-other@test.com")
	key := createTestAPIKey(t, plugin, owner, "default", nil, nil)
	apiKey := findTestAPIKey(t, plugin, key)

	err := plugin.Revoke(t.Context(), otherUser, apiKey.ID)

	require.ErrorIs(t, err, limen.ErrForbidden)
	assert.Equal(t, apiKey.ID, findTestAPIKey(t, plugin, key).ID)
}

func TestAPIKeyPlugin_Revoke_ReturnsNotFound(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAPIKeyPlugin(t)
	user := limen.SeedTestUser(t, l, "revoke-missing@test.com")

	err := plugin.Revoke(t.Context(), user, "missing")

	require.ErrorIs(t, err, limen.ErrRecordNotFound)
}

func TestAPIKeyPlugin_Rotate(t *testing.T) {
	t.Parallel()

	var sequence atomic.Int64
	l, plugin := newTestAPIKeyPlugin(t, Profile{
		ID:            "rotating",
		PrincipalType: PrincipalTypeUser,
		Prefix:        "rotating_",
		KeyGenerator: func(*Profile) string {
			return fmt.Sprintf("rotating_secret_%04d", sequence.Add(1))
		},
	})
	user := limen.SeedTestUser(t, l, "rotate@test.com")
	oldKey := createTestAPIKey(
		t,
		plugin,
		user,
		"rotating",
		Permissions{"file": {"read"}},
		nil,
	)
	oldAPIKey := findTestAPIKey(t, plugin, oldKey)
	expiresIn := int64(60)
	beforeRotate := time.Now()

	result, err := plugin.Rotate(t.Context(), user, oldAPIKey.ID, &ApiKeyRotateRequest{
		ExpiresIn:   &expiresIn,
		Permissions: Permissions{"file": {"write"}},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEqual(t, oldKey, result.Key)
	assert.Equal(t, oldAPIKey.ID, result.ID)
	assert.Equal(t, oldAPIKey.Name, result.Name)
	assert.Equal(t, oldAPIKey.Profile, result.Profile)
	assert.Equal(t, map[string][]string{"file": {"write"}}, result.Permissions)
	assert.Equal(t, plugin.hashAPIKey(result.Key), result.KeyHash)
	assert.Equal(t, result.Key[len(result.Key)-4:], result.Last4)
	require.NotNil(t, result.ExpiresAt)
	assert.WithinRange(t, *result.ExpiresAt, beforeRotate.Add(time.Minute), time.Now().Add(time.Minute))

	oldVerified, err := plugin.Verify(t.Context(), oldKey, nil)
	require.ErrorIs(t, err, ErrInvalidAPIKey)
	assert.Nil(t, oldVerified)

	newVerified, err := plugin.Verify(t.Context(), result.Key, nil)
	require.NoError(t, err)
	assert.Equal(t, result.ID, newVerified.ID)
}

func TestAPIKeyPlugin_Rotate_EnforcesOwnership(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAPIKeyPlugin(t)
	owner := limen.SeedTestUser(t, l, "rotate-owner@test.com")
	otherUser := limen.SeedTestUser(t, l, "rotate-other@test.com")
	key := createTestAPIKey(t, plugin, owner, "default", nil, nil)
	apiKey := findTestAPIKey(t, plugin, key)

	result, err := plugin.Rotate(t.Context(), otherUser, apiKey.ID, &ApiKeyRotateRequest{})

	require.ErrorIs(t, err, limen.ErrForbidden)
	assert.Nil(t, result)
	assert.Equal(t, apiKey.KeyHash, findTestAPIKey(t, plugin, key).KeyHash)
}

func TestAPIKeyPlugin_Rotate_ReturnsNotFound(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAPIKeyPlugin(t)
	user := limen.SeedTestUser(t, l, "rotate-missing@test.com")

	result, err := plugin.Rotate(t.Context(), user, "missing", &ApiKeyRotateRequest{})

	require.ErrorIs(t, err, limen.ErrRecordNotFound)
	assert.Nil(t, result)
}

type testPrincipalResolver struct {
	grantable Permissions
}

func (r *testPrincipalResolver) ResolvePrincipalID(
	_ context.Context,
	_ string,
	userID any,
) (any, error) {
	return userID, nil
}

func (r *testPrincipalResolver) GrantablePermissions(
	_ context.Context,
	_ string,
	_ any,
) (map[string][]string, error) {
	return r.grantable, nil
}
