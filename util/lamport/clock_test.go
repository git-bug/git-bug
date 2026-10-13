package lamport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testClock(t *testing.T, c Clock) {
	assertTime := func(expected Time) {
		t.Helper()
		actual, err := c.Time()
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	}

	assertTime(1)

	val, err := c.Increment()
	assert.NoError(t, err)
	assert.Equal(t, Time(2), val)
	assertTime(2)

	err = c.Witness(42)
	assert.NoError(t, err)
	assertTime(42)

	err = c.Witness(42)
	assert.NoError(t, err)
	assertTime(42)

	err = c.Witness(30)
	assert.NoError(t, err)
	assertTime(42)
}
