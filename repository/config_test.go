package repository

import (
	"testing"
	"time"

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
