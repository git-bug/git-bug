package test

import (
	"errors"
	"math/rand"
	"testing"
	"time"
)

type flaky struct {
	t testing.TB
	o *FlakyOptions
}

type FlakyOptions struct {
	InitialBackoff time.Duration
	MaxAttempts    int
	Jitter         float64
}

func NewFlaky(t testing.TB, o *FlakyOptions) *flaky {
	if o.InitialBackoff <= 0 {
		o.InitialBackoff = 500 * time.Millisecond
	}

	if o.MaxAttempts <= 0 {
		o.MaxAttempts = 3
	}

	if o.Jitter < 0 {
		o.Jitter = 0
	}

	return &flaky{t: t, o: o}
}

func (f *flaky) Run(fn func(t testing.TB)) {
	var last error

	for attempt := 1; attempt <= f.o.MaxAttempts; attempt++ {
		var failed bool

		fn(&recorder{
			TB:    f.t,
			fail:  func(e string) { failed = true; last = errors.New(e) },
			fatal: func(e string) { failed = true; last = errors.New(e) },
		})

		if !failed {
			return
		}

		if attempt < f.o.MaxAttempts {
			backoff := f.o.InitialBackoff * time.Duration(1<<uint(attempt-1))
			time.Sleep(applyJitter(backoff, f.o.Jitter))
		}
	}

	f.t.Fatalf("[%s] test failed after %d attempts: %s", f.t.Name(), f.o.MaxAttempts, last)
}

func applyJitter(d time.Duration, jitter float64) time.Duration {
	if jitter == 0 {
		return d
	}
	maxJitter := float64(d) * jitter
	delta := maxJitter * (rand.Float64()*2 - 1)
	return time.Duration(float64(d) + delta)
}
