package interrupt

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegisterAndErrorAtCleaning tests if the registered order was kept by checking the returned errors
func TestRegisterAndErrorAtCleaning(t *testing.T) {
	handlerCreated = true // this prevents goroutine from being started during the tests

	f1 := func() error {
		return errors.New("1")
	}
	f2 := func() error {
		return errors.New("2")
	}
	f3 := func() error {
		return nil
	}

	RegisterCleaner(f1)
	RegisterCleaner(f2)
	RegisterCleaner(f3)

	errl := clean()

	require.Len(t, errl, 2)

	// cleaners should execute in the reverse order they have been defined
	assert.Equal(t, "2", errl[0].Error())
	assert.Equal(t, "1", errl[1].Error())
}

func TestRegisterAndClean(t *testing.T) {
	handlerCreated = true // this prevents goroutine from being started during the tests

	f1 := func() error {
		return nil
	}
	f2 := func() error {
		return nil
	}

	RegisterCleaner(f1)
	RegisterCleaner(f2)

	errl := clean()
	assert.Len(t, errl, 0)
}

func TestCancel(t *testing.T) {
	handlerCreated = true // this prevents goroutine from being started during the tests

	f1 := func() error {
		return errors.New("1")
	}
	f2 := func() error {
		return errors.New("2")
	}

	cancel1 := RegisterCleaner(f1)
	RegisterCleaner(f2)

	cancel1()

	errl := clean()
	require.Len(t, errl, 1)

	assert.Equal(t, "2", errl[0].Error())
}
