package test

import (
	"testing"
	"time"
)

func Test_SucceedsImmediately(t *testing.T) {
	var attempts int

	f := NewFlaky(t, &FlakyOptions{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
	})

	f.Run(func(t testing.TB) {
		attempts++
		if attempts > 1 {
			t.Fatalf("should not retry on success")
		}
	})
}

func Test_EventualSuccess(t *testing.T) {
	var attempts int

	f := NewFlaky(t, &FlakyOptions{
		MaxAttempts:    5,
		InitialBackoff: 10 * time.Millisecond,
	})

	f.Run(func(t testing.TB) {
		attempts++
		if attempts < 3 {
			t.Fatalf("intentional failure")
		}
	})

	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}
