package graphql

import (
	"io"
	"sync"
)

// Defer will begin executing the given function and immediately return a result that will block until the function completes
func Defer(f func() Marshaler) Marshaler {
	var deferred deferred
	deferred.mu.Lock()

	go func() {
		deferred.result = f()
		deferred.mu.Unlock()
	}()

	return &deferred
}

type deferred struct {
	result Marshaler
	mu     sync.Mutex
}

func (d *deferred) MarshalGQL(w io.Writer) {
	d.mu.Lock()
	d.result.MarshalGQL(w)
	d.mu.Unlock()
}
