package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
