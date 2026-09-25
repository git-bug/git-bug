package lamport

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemClock(t *testing.T) {
	c := NewMemClock()
	testClock(t, c)
}

func TestMemClockOverflow(t *testing.T) {
	c := NewMemClockWithTime(math.MaxUint64 - 1)

	value, err := c.Increment()
	require.NoError(t, err)
	require.Equal(t, Time(math.MaxUint64), value)

	_, err = c.Increment()
	require.ErrorIs(t, err, ErrClockOverflow)

	value, err = c.Time()
	require.NoError(t, err)
	require.Equal(t, Time(math.MaxUint64), value)
}
