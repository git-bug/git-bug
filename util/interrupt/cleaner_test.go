package interrupt

import (
	"errors"
	"testing"
)

// TestRegisterAndErrorAtCleaning tests if the registered order was kept by checking the returned errors
func TestRegisterAndErrorAtCleaning(t *testing.T) {
	active = true // this prevents goroutine from being started during the tests

	f := func() error {
		return errors.New("X")
	}
	f2 := func() error {
		return errors.New("Y")
	}
	f3 := func() error {
		return nil
	}
	RegisterCleaner(f)
	RegisterCleaner(f2, f3)
	// count := 0

	errl := Clean()
	if len(errl) != 2 {
		t.Fatalf("unexpected error count")
	}
	if errl[0].Error() != "Y" && errl[1].Error() != "X" {
		t.Fatalf("unexpected error order")

	}
}

func TestRegisterAndClean(t *testing.T) {
	active = true // this prevents goroutine from being started during the tests

	f := func() error {
		return nil
	}
	f2 := func() error {
		return nil
	}
	RegisterCleaner(f, f2)

	errl := Clean()
	if len(errl) != 0 {
		t.Fatalf("unexpected error count")
	}
}
