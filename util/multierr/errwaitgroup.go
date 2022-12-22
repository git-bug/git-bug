package multierr

import (
	"context"
	"fmt"
	"sync"
)

type token struct{}

// A ErrWaitGroup is a collection of goroutines working on subtasks that are part of
// the same overall task.
//
// A zero ErrWaitGroup is valid, has no limit on the number of active goroutines,
// and does not cancel on error.
type ErrWaitGroup struct {
	cancel func()

	wg sync.WaitGroup

	sem chan token

	mu  sync.Mutex
	err error
}

func (g *ErrWaitGroup) done() {
	if g.sem != nil {
		<-g.sem
	}
	g.wg.Done()
}

// WithContext returns a new ErrWaitGroup and an associated Context derived from ctx.
//
// The derived Context is canceled the first time Wait returns.
func WithContext(ctx context.Context) (*ErrWaitGroup, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &ErrWaitGroup{cancel: cancel}, ctx
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the combined non-nil errors (if any) from them.
func (g *ErrWaitGroup) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.err
}

// Go calls the given function in a new goroutine.
// It blocks until the new goroutine can be added without the number of
// active goroutines in the group exceeding the configured limit.
func (g *ErrWaitGroup) Go(f func() error) {
	if g.sem != nil {
		g.sem <- token{}
	}

	g.wg.Add(1)
	go func() {
		defer g.done()

		if err := f(); err != nil {
			g.mu.Lock()
			g.err = Join(g.err, err)
			g.mu.Unlock()
		}
	}()
}

// TryGo calls the given function in a new goroutine only if the number of
// active goroutines in the group is currently below the configured limit.
//
// The return value reports whether the goroutine was started.
func (g *ErrWaitGroup) TryGo(f func() error) bool {
	if g.sem != nil {
		select {
		case g.sem <- token{}:
			// Note: this allows barging iff channels in general allow barging.
		default:
			return false
		}
	}

	g.wg.Add(1)
	go func() {
		defer g.done()

		if err := f(); err != nil {
			g.mu.Lock()
			err = Join(g.err, err)
			g.mu.Unlock()
		}
	}()
	return true
}

// SetLimit limits the number of active goroutines in this group to at most n.
// A negative value indicates no limit.
//
// Any subsequent call to the Go method will block until it can add an active
// goroutine without exceeding the configured limit.
//
// The limit must not be modified while any goroutines in the group are active.
func (g *ErrWaitGroup) SetLimit(n int) {
	if n < 0 {
		g.sem = nil
		return
	}
	if len(g.sem) != 0 {
		panic(fmt.Errorf("errwaitgroup: modify limit while %v goroutines in the group are still active", len(g.sem)))
	}
	g.sem = make(chan token, n)
}
