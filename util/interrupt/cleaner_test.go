package interrupt

import (
	"testing"
)

func TestRegister(t *testing.T) {
	active = true // this prevents goroutine from being started during the tests

	f := func() error {
		return nil
	}
	f2 := func() error {
		return nil
	}
	f3 := func() error {
		return nil
	}
	RegisterCleaner(f)
	RegisterCleaner(f2, f3)
	count := 0
	for _, fn := range cleaners {
		errt := fn()
		count++
		if errt != nil {
			t.Fatalf("bad err value")
		}
	}
	if count != 3 {
		t.Fatalf("different number of errors")
	}

}
