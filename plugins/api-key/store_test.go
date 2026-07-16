package apikey

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestAPIKeyStore_CacheRoundTrip(t *testing.T) {
	t.Parallel()

	plugin := New()
	l, _ := limen.NewTestLimen(t, plugin)
	user := limen.SeedTestUser(t, l, "cache-round-trip@test.com")
	expiresIn := int64(60)
	key := createTestAPIKey(
		t,
		plugin,
		user,
		"default",
		Permissions{"file": {"read"}},
		&expiresIn,
	)
	keyHash := plugin.hashAPIKey(key)

	fromDatabase, err := plugin.store.FindOne(t.Context(), keyHash, false)
	require.NoError(t, err)
	fromCache, err := plugin.store.FindOne(t.Context(), keyHash, true)
	require.NoError(t, err)

	assert.EqualValues(t, fromDatabase.ID, fromCache.ID)
	assert.EqualValues(t, fromDatabase.CreatedByUserID, fromCache.CreatedByUserID)
	assert.Equal(t, fromDatabase.PrincipalType, fromCache.PrincipalType)
	assert.EqualValues(t, fromDatabase.PrincipalID, fromCache.PrincipalID)
	assert.Equal(t, fromDatabase.KeyHash, fromCache.KeyHash)
	assert.Equal(t, fromDatabase.Permissions, fromCache.Permissions)
	assert.Equal(t, fromDatabase.Enabled, fromCache.Enabled)
	require.NotNil(t, fromCache.ExpiresAt)
	assert.WithinDuration(t, *fromDatabase.ExpiresAt, *fromCache.ExpiresAt, time.Nanosecond)
}
