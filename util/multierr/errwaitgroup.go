package multierr

import (
	"errors"
	"sync"
)

// A ErrWaitGroup is a collection of goroutines working on subtasks that are part of
// the same overall task. Unlike errgroup.Group, it collects every error instead of
// only the first one.
//
// A zero ErrWaitGroup is valid.
type ErrWaitGroup struct {
	wg sync.WaitGroup

	mu  sync.Mutex
	err error
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the combined non-nil errors (if any) from them.
func (g *ErrWaitGroup) Wait() error {
	g.wg.Wait()
	return g.err
}

// Go calls the given function in a new goroutine.
func (g *ErrWaitGroup) Go(f func() error) {
	g.wg.Go(func() {
		if err := f(); err != nil {
			g.mu.Lock()
			g.err = errors.Join(g.err, err)
			g.mu.Unlock()
		}
	})
}
