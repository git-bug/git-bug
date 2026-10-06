package repository

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// The change sources of a GoGitRepo look at the loose refs of each namespace,
// refs/<namespace>/, and at packed-refs, in the git directory where GoGitRepo
// reads them, see listRefs.

// watchWindow is how long Watch gathers events into a single Change: a fetch
// rewriting a hundred refs is one Change.
const watchWindow = 50 * time.Millisecond

// pollGranularity is the coarsest mtime resolution Poll expects of a
// filesystem, see pollSource.
const pollGranularity = 2 * time.Second

// NewWatchSource returns a source reporting the changes of the refs of repo as
// they happen, with the filesystem's notifications. A loose ref names its key,
// a rewrite of packed-refs names nothing.
//
// It sees nothing that another host writes on a network filesystem, and isn't
// suitable for platforms where watching a directory takes a file descriptor
// per entry (macOS, BSDs), see NewAutoSource.
func NewWatchSource(repo *GoGitRepo) ChangeSource {
	return watchSource{gitDir: repo.path}
}

// NewPollSource returns a source checking, every period, whether the refs of
// repo changed, from the stat of the directory of each namespace and of
// packed-refs. It works on every platform and filesystem, but only sees refs
// that are renamed into place, as git and GoGitRepo write them: a ref
// rewritten in place, as released go-git does, leaves its directory's mtime
// unchanged.
func NewPollSource(repo *GoGitRepo, period time.Duration) ChangeSource {
	return pollSource{gitDir: repo.path, period: period}
}

// NewAutoSource returns a source watching the refs of repo where watching
// works, see NewWatchSource, and polling them every pollPeriod otherwise, see
// NewPollSource. A watch failing, when subscribing or later on, falls back to
// polling for the life of the subscription.
func NewAutoSource(repo *GoGitRepo, pollPeriod time.Duration) ChangeSource {
	return autoSource{
		watch:    watchSource{gitDir: repo.path},
		poll:     pollSource{gitDir: repo.path, period: pollPeriod},
		canWatch: canWatch(repo.path),
	}
}

type watchSource struct {
	gitDir string
}

// Subscribe reports the events of the watcher until ctx is done, or until the
// watcher fails. Events are gathered for watchWindow, and while the channel
// isn't read.
func (s watchSource) Subscribe(ctx context.Context, namespaces []string) (<-chan Change, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("watching for changes: %w", err)
	}

	refsDir := filepath.Join(s.gitDir, "refs")
	byDir := make(map[string]string, len(namespaces))
	for _, namespace := range namespaces {
		byDir[filepath.Join(refsDir, namespace)] = namespace
	}

	// the git directory for packed-refs, refs for the namespace directories
	// to appear, as watches are not recursive
	paths := []string{s.gitDir, refsDir}
	for dir := range byDir {
		paths = append(paths, dir)
	}
	for _, path := range paths {
		err = w.Add(path)
		if errors.Is(err, fs.ErrNotExist) && path != refsDir && path != s.gitDir {
			// no loose ref yet: added once it appears
			continue
		}
		if err != nil {
			_ = w.Close()
			return nil, fmt.Errorf("watching %s for changes: %w", path, err)
		}
	}

	out := make(chan Change)
	go func() {
		defer close(out)
		defer func() { _ = w.Close() }()

		// what was gathered since the last change sent: every ref, or the
		// keys of each namespace
		var pendingAll bool
		var pendingKeys map[string]map[string]struct{}
		// window fires when what was gathered can be sent, nil if nothing was
		var window <-chan time.Time
		var ready bool

		for {
			// send is enabled once the window passed, with what was gathered
			// until this iteration
			var send chan<- Change
			var change Change
			if ready {
				send = out
				if !pendingAll {
					change.Keys = make(map[string][]string, len(pendingKeys))
					for namespace, keys := range pendingKeys {
						change.Keys[namespace] = slices.Collect(maps.Keys(keys))
					}
				}
			}

			select {
			case <-ctx.Done():
				return

			case event, ok := <-w.Events:
				if !ok {
					return
				}
				if event.Op == fsnotify.Chmod {
					continue
				}
				dir, name := filepath.Dir(event.Name), filepath.Base(event.Name)

				switch dir {
				case s.gitDir:
					if name != "packed-refs" {
						continue
					}
					pendingAll = true

				case refsDir:
					if _, ok := byDir[event.Name]; !ok {
						continue
					}
					// Windows and kqueue also report a write of a namespace
					// directory when its entries change: those changes come,
					// named, from the watch of the namespace directory itself
					if !event.Has(fsnotify.Create | fsnotify.Remove | fsnotify.Rename) {
						continue
					}
					// a namespace directory appeared, or was removed or
					// replaced, with whatever refs it holds
					if event.Has(fsnotify.Create) {
						err := w.Add(event.Name)
						if err != nil && !errors.Is(err, fs.ErrNotExist) {
							return
						}
					}
					pendingAll = true

				default:
					namespace, ok := byDir[dir]
					if !ok || strings.HasSuffix(name, refLockSuffix) {
						continue
					}
					if pendingKeys == nil {
						pendingKeys = make(map[string]map[string]struct{})
					}
					if pendingKeys[namespace] == nil {
						pendingKeys[namespace] = make(map[string]struct{})
					}
					pendingKeys[namespace][name] = struct{}{}
				}

				if !ready && window == nil {
					window = time.After(watchWindow)
				}

			case err, ok := <-w.Errors:
				if !ok || !errors.Is(err, fsnotify.ErrEventOverflow) {
					return
				}
				// events were dropped: anything may have changed
				pendingAll = true
				if !ready && window == nil {
					window = time.After(watchWindow)
				}

			case <-window:
				window = nil
				ready = true

			case send <- change:
				pendingAll = false
				pendingKeys = nil
				ready = false
			}
		}
	}()

	return out, nil
}

// pollSource reports a change when the stat of the directory of a namespace or
// of packed-refs differs from the previous poll.
//
// A filesystem may record mtimes with a coarse resolution: a path written again
// within it keeps the same mtime. A stat whose mtime is within pollGranularity
// of the poll is therefore not trusted, and reported as a change at the next
// poll.
type pollSource struct {
	gitDir string
	period time.Duration
}

// pathState is the stat of a path at a poll.
type pathState struct {
	info os.FileInfo // nil if the path doesn't exist
	// recent is set if its mtime was too close to the poll to be trusted
	recent bool
}

func (s pollSource) Subscribe(ctx context.Context, namespaces []string) (<-chan Change, error) {
	refsDir := filepath.Join(s.gitDir, "refs")
	info, err := os.Stat(refsDir)
	if err != nil {
		return nil, fmt.Errorf("polling for changes: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("polling for changes: %s is not a directory", refsDir)
	}

	paths := []string{filepath.Join(s.gitDir, "packed-refs")}
	for _, namespace := range namespaces {
		paths = append(paths, filepath.Join(refsDir, namespace))
	}

	previous, err := s.stat(paths)
	if err != nil {
		return nil, fmt.Errorf("polling for changes: %w", err)
	}

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

			current, err := s.stat(paths)
			if err != nil {
				return
			}
			changed := false
			for i := range paths {
				a, b := previous[i].info, current[i].info
				same := a == nil && b == nil ||
					a != nil && b != nil && os.SameFile(a, b) && a.ModTime().Equal(b.ModTime()) && a.Size() == b.Size()
				if previous[i].recent || !same {
					changed = true
					break
				}
			}
			previous = current
			if !changed {
				continue
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

func (s pollSource) stat(paths []string) ([]pathState, error) {
	now := time.Now()
	states := make([]pathState, len(paths))
	for i, path := range paths {
		info, err := os.Stat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		states[i] = pathState{
			info:   info,
			recent: now.Sub(info.ModTime()).Abs() < pollGranularity,
		}
	}
	return states, nil
}

type autoSource struct {
	watch    ChangeSource
	poll     ChangeSource
	canWatch bool
}

func (s autoSource) Subscribe(ctx context.Context, namespaces []string) (<-chan Change, error) {
	if !s.canWatch {
		return s.poll.Subscribe(ctx, namespaces)
	}

	watched, err := s.watch.Subscribe(ctx, namespaces)
	if err != nil {
		return s.poll.Subscribe(ctx, namespaces)
	}

	out := make(chan Change)
	go func() {
		defer close(out)

		forward := func(in <-chan Change) {
			for change := range in {
				select {
				case <-ctx.Done():
				case out <- change:
				}
			}
		}

		forward(watched)
		if ctx.Err() != nil {
			return
		}

		// The watch failed for good, possibly having missed changes. It isn't
		// retried, so as not to oscillate.
		polled, err := s.poll.Subscribe(ctx, namespaces)
		if err != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case out <- Change{}:
		}
		forward(polled)
	}()

	return out, nil
}
