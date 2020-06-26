package lamport

import "testing"

func TestMemClock(t *testing.T) {
	c := NewMemClock()
	testClock(t, c)
}
