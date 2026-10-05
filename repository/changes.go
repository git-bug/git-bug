package repository

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Change reports that entities may have been modified.
//
// It is a hint and never exhaustive: a source may drop events, and some changes
// (a packed-refs rewrite, a dropped-event overflow) name nothing at all. An
// empty Change therefore means "something happened, unknown what" and obliges
// the consumer to do a full comparison; it never means "nothing happened". Nor
// does it distinguish a deletion from a movement. A consumer may use it to do
// eager work, but must never conclude that unnamed entities are unchanged:
// completeness comes from Periodic, not from this field.
type Change struct {
	// Keys of the refs the source believes changed, by namespace ("bugs",
	// "identities"). Empty means "something changed, but not what".
	Keys map[string][]string
}

// ChangeSource notifies that the repository may have been modified, possibly by
// another process.
//
// A source may coalesce, duplicate or delay notifications. Only Periodic is
// guaranteed not to miss a change; every other source may drop them, and may
// stop working entirely.
type ChangeSource interface {
	// Subscribe returns a channel receiving a Change whenever the refs of the
	// given namespaces may have changed. It is closed when ctx is done or when
	// the source stops permanently. An error means this source cannot work
	// here; the caller should carry on without it.
	Subscribe(ctx context.Context, namespaces []string) (<-chan Change, error)
}

// Periodic returns a source sending an empty Change every period. It assumes
// nothing about how the repository is written, so it is the one source that
// can't miss a change: it is the backstop of the others.
func Periodic(period time.Duration) ChangeSource {
	return periodicSource{period: period}
}

type periodicSource struct {
	period time.Duration
}

func (s periodicSource) Subscribe(ctx context.Context, _ []string) (<-chan Change, error) {
	out := make(chan Change)
	go func() {
		defer close(out)
		ticker := time.NewTicker(s.period)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			select {
			case <-ctx.Done():
				return
			case out <- Change{}:
			}
		}
	}()
	return out, nil
}

// Merge returns a source forwarding what any of sources reports. Its channel is
// closed once all of theirs are.
//
// Subscribe fails only if every source failed. If some did, it returns both the
// channel of the others and the errors of those that failed.
func Merge(sources ...ChangeSource) ChangeSource {
	return mergedSource(sources)
}

type mergedSource []ChangeSource

func (m mergedSource) Subscribe(ctx context.Context, namespaces []string) (<-chan Change, error) {
	var chans []<-chan Change
	var errs []error
	for _, source := range m {
		ch, err := source.Subscribe(ctx, namespaces)
		if err != nil {
			errs = append(errs, err)
		}
		if ch != nil {
			chans = append(chans, ch)
		}
	}
	if len(chans) == 0 {
		if len(errs) == 0 {
			errs = append(errs, errors.New("no change source"))
		}
		return nil, errors.Join(errs...)
	}

	out := make(chan Change)
	var wg sync.WaitGroup
	for _, ch := range chans {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for change := range ch {
				select {
				case <-ctx.Done():
					// drain, for the source to see ctx and close
				case out <- change:
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()

	return out, errors.Join(errs...)
}
