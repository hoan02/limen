package redis

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func setupTestAdapter(t *testing.T) (*Adapter, *miniredis.Miniredis) {
	t.Helper()

	server := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})

	return New(client), server
}

func TestAdapter_SetAndGet(t *testing.T) {
	t.Parallel()

	adapter, _ := setupTestAdapter(t)
	ctx := t.Context()
	value := []byte{0, 1, 2, 255}

	require.NoError(t, adapter.Set(ctx, "key", value, 5*time.Minute))

	got, err := adapter.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, value, got)
}

func TestAdapter_GetNotFound(t *testing.T) {
	t.Parallel()

	adapter, _ := setupTestAdapter(t)

	_, err := adapter.Get(t.Context(), "missing")
	assert.ErrorIs(t, err, limen.ErrRecordNotFound)
}

func TestAdapter_GetExpired(t *testing.T) {
	t.Parallel()

	adapter, server := setupTestAdapter(t)
	ctx := t.Context()

	require.NoError(t, adapter.Set(ctx, "key", []byte("value"), time.Minute))
	server.FastForward(time.Minute)

	_, err := adapter.Get(ctx, "key")
	assert.ErrorIs(t, err, limen.ErrRecordNotFound)
}

func TestAdapter_SetWithoutExpiry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{name: "zero", ttl: 0},
		{name: "negative", ttl: -time.Second},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			adapter, server := setupTestAdapter(t)
			ctx := t.Context()

			require.NoError(t, adapter.Set(ctx, "key", []byte("value"), test.ttl))
			server.FastForward(24 * time.Hour)

			got, err := adapter.Get(ctx, "key")
			require.NoError(t, err)
			assert.Equal(t, []byte("value"), got)
		})
	}
}

func TestAdapter_SetOverwritesValueAndExpiry(t *testing.T) {
	t.Parallel()

	adapter, server := setupTestAdapter(t)
	ctx := t.Context()

	require.NoError(t, adapter.Set(ctx, "key", []byte("old"), time.Minute))
	require.NoError(t, adapter.Set(ctx, "key", []byte("new"), 0))
	server.FastForward(2 * time.Minute)

	got, err := adapter.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, []byte("new"), got)
}

func TestAdapter_Has(t *testing.T) {
	t.Parallel()

	adapter, server := setupTestAdapter(t)
	ctx := t.Context()

	exists, err := adapter.Has(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, adapter.Set(ctx, "key", []byte("value"), time.Minute))
	exists, err = adapter.Has(ctx, "key")
	require.NoError(t, err)
	assert.True(t, exists)

	server.FastForward(time.Minute)
	exists, err = adapter.Has(ctx, "key")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestAdapter_Delete(t *testing.T) {
	t.Parallel()

	adapter, _ := setupTestAdapter(t)
	ctx := t.Context()

	require.NoError(t, adapter.Set(ctx, "key", []byte("value"), 0))
	require.NoError(t, adapter.Delete(ctx, "key"))
	require.NoError(t, adapter.Delete(ctx, "key"))

	_, err := adapter.Get(ctx, "key")
	assert.ErrorIs(t, err, limen.ErrRecordNotFound)
}

func TestAdapter_SetExpiry(t *testing.T) {
	t.Parallel()

	adapter, server := setupTestAdapter(t)
	ctx := t.Context()

	require.NoError(t, adapter.Set(ctx, "key", []byte("value"), 0))
	require.NoError(t, adapter.SetExpiry(ctx, "key", time.Minute))
	server.FastForward(time.Minute)

	_, err := adapter.Get(ctx, "key")
	assert.ErrorIs(t, err, limen.ErrRecordNotFound)
}

func TestAdapter_SetExpiryNonPositivePersistsValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{name: "zero", ttl: 0},
		{name: "negative", ttl: -time.Second},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			adapter, server := setupTestAdapter(t)
			ctx := t.Context()

			require.NoError(t, adapter.Set(ctx, "key", []byte("value"), time.Minute))
			require.NoError(t, adapter.SetExpiry(ctx, "key", test.ttl))
			server.FastForward(2 * time.Minute)

			got, err := adapter.Get(ctx, "key")
			require.NoError(t, err)
			assert.Equal(t, []byte("value"), got)
		})
	}
}

func TestAdapter_SetExpiryMissingKeyIsNoOp(t *testing.T) {
	t.Parallel()

	adapter, _ := setupTestAdapter(t)

	assert.NoError(t, adapter.SetExpiry(t.Context(), "missing", time.Minute))
}

func TestAdapter_IncrementAndDecrement(t *testing.T) {
	t.Parallel()

	adapter, _ := setupTestAdapter(t)
	ctx := t.Context()

	value, err := adapter.Increment(ctx, "counter", 5)
	require.NoError(t, err)
	assert.Equal(t, int64(5), value)

	value, err = adapter.Decrement(ctx, "counter", 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3), value)

	got, err := adapter.Get(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, []byte("3"), got)
}

func TestAdapter_IncrementIsAtomic(t *testing.T) {
	t.Parallel()

	adapter, _ := setupTestAdapter(t)
	ctx := t.Context()
	errs := make(chan error, 100)

	var group sync.WaitGroup
	for range 100 {
		group.Go(func() {
			_, err := adapter.Increment(ctx, "counter", 1)
			errs <- err
		})
	}
	group.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	got, err := adapter.Get(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, []byte("100"), got)
}

func TestAdapter_IncrementPreservesExpiry(t *testing.T) {
	t.Parallel()

	adapter, server := setupTestAdapter(t)
	ctx := t.Context()

	require.NoError(t, adapter.Set(ctx, "counter", []byte("1"), time.Minute))
	_, err := adapter.Increment(ctx, "counter", 1)
	require.NoError(t, err)
	server.FastForward(time.Minute)

	_, err = adapter.Get(ctx, "counter")
	assert.ErrorIs(t, err, limen.ErrRecordNotFound)
}

func TestAdapter_IncrementRejectsNonIntegerValue(t *testing.T) {
	t.Parallel()

	adapter, _ := setupTestAdapter(t)
	ctx := t.Context()

	require.NoError(t, adapter.Set(ctx, "counter", []byte("not-an-integer"), 0))
	_, err := adapter.Increment(ctx, "counter", 1)
	assert.Error(t, err)
}

func TestAdapter_PropagatesRedisErrors(t *testing.T) {
	t.Parallel()

	server := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	adapter := New(client)
	require.NoError(t, client.Close())

	_, err := adapter.Get(context.Background(), "key")
	require.Error(t, err)
	assert.False(t, errors.Is(err, limen.ErrRecordNotFound))
}
