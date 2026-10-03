package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergedConfig(t *testing.T) {
	local := NewMemConfig()
	global := NewMemConfig()
	merged := mergeConfig(local, global)

	require.NoError(t, global.StoreBool("bool", true))
	require.NoError(t, global.StoreString("string", "foo"))
	require.NoError(t, global.StoreTimestamp("timestamp", time.Unix(1234, 0)))

	val1, err := merged.ReadBool("bool")
	require.NoError(t, err)
	require.Equal(t, val1, true)

	val2, err := merged.ReadString("string")
	require.NoError(t, err)
	require.Equal(t, val2, "foo")

	val3, err := merged.ReadTimestamp("timestamp")
	require.NoError(t, err)
	require.Equal(t, val3, time.Unix(1234, 0))

	require.NoError(t, local.StoreBool("bool", false))
	require.NoError(t, local.StoreString("string", "bar"))
	require.NoError(t, local.StoreTimestamp("timestamp", time.Unix(5678, 0)))

	val1, err = merged.ReadBool("bool")
	require.NoError(t, err)
	require.Equal(t, val1, false)

	val2, err = merged.ReadString("string")
	require.NoError(t, err)
	require.Equal(t, val2, "bar")

	val3, err = merged.ReadTimestamp("timestamp")
	require.NoError(t, err)
	require.Equal(t, val3, time.Unix(5678, 0))

	all, err := merged.ReadAll("")
	require.NoError(t, err)
	require.Equal(t, all, map[string]string{
		"bool":      "false",
		"string":    "bar",
		"timestamp": "5678",
	})
}

func TestGetDefaultString(t *testing.T) {
	cfg := NewMemConfig()

	// Test with missing key - should return default
	val, err := GetDefaultString("missing.key", cfg, "default_value")
	require.NoError(t, err)
	assert.Equal(t, "default_value", val)

	// Test with existing key - should return actual value
	require.NoError(t, cfg.StoreString("existing.key", "actual_value"))
	val, err = GetDefaultString("existing.key", cfg, "default_value")
	require.NoError(t, err)
	assert.Equal(t, "actual_value", val)

	// Test with empty string value - should return empty string, not default
	require.NoError(t, cfg.StoreString("empty.key", ""))
	val, err = GetDefaultString("empty.key", cfg, "default_value")
	require.NoError(t, err)
	assert.Equal(t, "", val)

	// Test the specific git-bug.remote case
	val, err = GetDefaultString("git-bug.remote", cfg, "origin")
	require.NoError(t, err)
	assert.Equal(t, "origin", val)

	require.NoError(t, cfg.StoreString("git-bug.remote", "upstream"))
	val, err = GetDefaultString("git-bug.remote", cfg, "origin")
	require.NoError(t, err)
	assert.Equal(t, "upstream", val)
}

func TestParseTimestamp(t *testing.T) {
	// RFC3339Nano with fractions
	ts, err := ParseTimestamp("2026-10-01T14:13:21.123456789Z")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 1, 14, 13, 21, 123456789, time.UTC), ts)

	// RFC3339 without fractions
	ts, err = ParseTimestamp("2026-10-01T14:13:21Z")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 1, 14, 13, 21, 0, time.UTC), ts)

	// Unix seconds
	ts, err = ParseTimestamp("1234567890")
	require.NoError(t, err)
	assert.Equal(t, time.Unix(1234567890, 0), ts)

	// Unix nanoseconds
	ts, err = ParseTimestamp("1727788401123456789")
	require.NoError(t, err)
	assert.Equal(t, time.Unix(0, 1727788401123456789), ts)

	// Invalid input
	_, err = ParseTimestamp("invalid-timestamp")
	require.Error(t, err)
}
