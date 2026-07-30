package sql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeWriteValue(t *testing.T) {
	t.Parallel()

	t.Run("leaves scalars unchanged", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		cases := []any{"text", true, 42, int64(7), 1.5, now, []byte("raw"), nil}
		for _, in := range cases {
			out, err := normalizeWriteValue(in)
			require.NoError(t, err)
			assert.Equal(t, in, out)
		}
	})

	t.Run("marshals maps and slices to JSON strings", func(t *testing.T) {
		t.Parallel()

		out, err := normalizeWriteValue(map[string]any{"active_organization_id": 23})
		require.NoError(t, err)
		assert.Equal(t, `{"active_organization_id":23}`, out)

		out, err = normalizeWriteValue([]string{"admin", "member"})
		require.NoError(t, err)
		assert.Equal(t, `["admin","member"]`, out)
	})
}

func TestNormalizeWriteMap(t *testing.T) {
	t.Parallel()

	in := map[string]any{
		"token":    "abc",
		"metadata": map[string]any{"ip": "::1"},
	}
	out, err := normalizeWriteMap(in)
	require.NoError(t, err)
	assert.Equal(t, "abc", out["token"])
	assert.Equal(t, `{"ip":"::1"}`, out["metadata"])
	assert.IsType(t, map[string]any{}, in["metadata"], "input map must not be mutated")
}
