package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

// testSource is a change source sending what the test sends on changes, or
// failing to subscribe if err is set.
type testSource struct {
	changes chan Change
	err     error
}

func newTestSource() *testSource {
	return &testSource{changes: make(chan Change)}
}

// Subscribe forwards the changes until ctx is done or changes is closed.
func (s *testSource) Subscribe(ctx context.Context, _ []string) (<-chan Change, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make(chan Change)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case change, ok := <-s.changes:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case out <- change:
				}
			}
		}
	}()
	return out, nil
}

// nextChange returns the next change of ch, failing if none comes in time.
func nextChange(t *testing.T, ch <-chan Change) Change {
	t.Helper()
	select {
	case change, ok := <-ch:
		require.True(t, ok, "closed")
		for _, keys := range change.Keys {
			slices.Sort(keys)
		}
		return change
	case <-time.After(5 * time.Second):
		require.FailNow(t, "no change reported")
		return Change{}
	}
}

// requireNoChange checks that ch reports nothing for a while.
func requireNoChange(t *testing.T, ch <-chan Change) {
	t.Helper()
	select {
	case change := <-ch:
		require.FailNow(t, "unexpected change", "%v", change)
	case <-time.After(300 * time.Millisecond):
	}
}

func named(namespace string, keys ...string) Change {
	slices.Sort(keys)
	return Change{Keys: map[string][]string{namespace: keys}}
}

func TestPeriodic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		ch, err := Periodic(time.Minute).Subscribe(ctx, nil)
		require.NoError(t, err)

		start := time.Now()
		for i := range 3 {
			require.Equal(t, Change{}, <-ch)
			require.Equal(t, time.Duration(i+1)*time.Minute, time.Since(start))
		}

		cancel()
		for range ch {
		}
	})
}

func TestMerge(t *testing.T) {
	t.Run("forwards every source, until all are closed", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			a, b := newTestSource(), newTestSource()
			ch, err := Merge(a, b).Subscribe(t.Context(), nil)
			require.NoError(t, err)

			a.changes <- named("bugs", "a")
			require.Equal(t, named("bugs", "a"), <-ch)
			b.changes <- named("bugs", "b")
			require.Equal(t, named("bugs", "b"), <-ch)

			close(a.changes)
			b.changes <- Change{}
			require.Equal(t, Change{}, <-ch)
			close(b.changes)
			_, ok := <-ch
			require.False(t, ok)
		})
	})

	t.Run("closes when ctx is done", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			ch, err := Merge(newTestSource(), Periodic(time.Minute)).Subscribe(ctx, nil)
			require.NoError(t, err)
			cancel()
			for range ch {
			}
		})
	})

	t.Run("some sources failing", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ok := newTestSource()
			failing := &testSource{err: errors.New("can't work here")}
			ch, err := Merge(failing, ok).Subscribe(t.Context(), nil)
			require.ErrorContains(t, err, "can't work here")
			require.NotNil(t, ch)

			ok.changes <- Change{}
			require.Equal(t, Change{}, <-ch)
			close(ok.changes)
			for range ch {
			}
		})
	})

	t.Run("every source failing", func(t *testing.T) {
		ch, err := Merge(&testSource{err: errors.New("one")}, &testSource{err: errors.New("two")}).Subscribe(t.Context(), nil)
		require.ErrorContains(t, err, "one")
		require.ErrorContains(t, err, "two")
		require.Nil(t, ch)
	})
}

// newWatchedRepo returns a repository with a ref in the bugs namespace, and
// commits to point refs to.
func newWatchedRepo(t *testing.T) (*GoGitRepo, []Hash) {
	t.Helper()
	repo := newTestGoGitRepo(t, false)
	commits := storeTestCommits(t, repo, 2)
	require.NoError(t, repo.UpdateRef("bugs", randomKey(), "", commits[0]))
	return repo, commits
}

func TestWatchSource(t *testing.T) {
	requireGitBinary(t)

	subscribe := func(t *testing.T, repo *GoGitRepo) <-chan Change {
		t.Helper()
		ch, err := NewWatchSource(repo).Subscribe(t.Context(), []string{"bugs", "identities"})
		require.NoError(t, err)
		return ch
	}

	for _, tc := range []struct {
		name string
		// change changes the repository, and returns what the source reports
		change func(t *testing.T, repo *GoGitRepo, commits []Hash) Change
	}{
		{
			name: "a ref written by GoGitRepo names its key",
			change: func(t *testing.T, repo *GoGitRepo, commits []Hash) Change {
				key := randomKey()
				require.NoError(t, repo.UpdateRef("bugs", key, "", commits[0]))
				return named("bugs", key)
			},
		},
		{
			name: "a ref written by the git binary names its key",
			change: func(t *testing.T, repo *GoGitRepo, commits []Hash) Change {
				key := randomKey()
				runGit(t, repo.path, "update-ref", "refs/bugs/"+key, commits[1].String())
				return named("bugs", key)
			},
		},
		{
			name: "refs written together are one change",
			change: func(t *testing.T, repo *GoGitRepo, commits []Hash) Change {
				keys := []string{randomKey(), randomKey(), randomKey()}
				for _, key := range keys {
					require.NoError(t, repo.UpdateRef("bugs", key, "", commits[0]))
				}
				return named("bugs", keys...)
			},
		},
		{
			name: "a rewrite of packed-refs names nothing",
			change: func(t *testing.T, repo *GoGitRepo, commits []Hash) Change {
				runGit(t, repo.path, "pack-refs", "--all")
				return Change{}
			},
		},
		{
			name: "a namespace appearing names nothing",
			change: func(t *testing.T, repo *GoGitRepo, commits []Hash) Change {
				require.NoError(t, repo.UpdateRef("identities", randomKey(), "", commits[0]))
				return Change{}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, commits := newWatchedRepo(t)
			ch := subscribe(t, repo)
			expected := tc.change(t, repo, commits)
			require.Equal(t, expected, nextChange(t, ch))
		})
	}

	t.Run("a removed ref names its key", func(t *testing.T) {
		repo, commits := newWatchedRepo(t)
		ch := subscribe(t, repo)

		key := randomKey()
		require.NoError(t, repo.UpdateRef("bugs", key, "", commits[0]))
		require.Equal(t, named("bugs", key), nextChange(t, ch))
		require.NoError(t, repo.RemoveRef("bugs", key))
		require.Equal(t, named("bugs", key), nextChange(t, ch))
	})

	t.Run("a namespace appearing is watched", func(t *testing.T) {
		repo, commits := newWatchedRepo(t)
		ch := subscribe(t, repo)

		require.NoError(t, repo.UpdateRef("identities", randomKey(), "", commits[0]))
		require.Equal(t, Change{}, nextChange(t, ch))

		key := randomKey()
		require.NoError(t, repo.UpdateRef("identities", key, "", commits[0]))
		require.Equal(t, named("identities", key), nextChange(t, ch))
	})

	t.Run("other refs are ignored", func(t *testing.T) {
		repo, commits := newWatchedRepo(t)
		ch := subscribe(t, repo)

		runGit(t, repo.path, "update-ref", "refs/heads/main", commits[0].String())
		runGit(t, repo.path, "update-ref", "refs/other/"+randomKey(), commits[0].String())
		require.NoError(t, repo.UpdateRef("elsewhere", randomKey(), "", commits[0]))
		requireNoChange(t, ch)
	})

	t.Run("closes when ctx is done", func(t *testing.T) {
		repo, _ := newWatchedRepo(t)
		ctx, cancel := context.WithCancel(t.Context())
		ch, err := NewWatchSource(repo).Subscribe(ctx, []string{"bugs"})
		require.NoError(t, err)
		cancel()
		for range ch {
		}
	})

	t.Run("fails without a refs directory", func(t *testing.T) {
		_, err := watchSource{gitDir: t.TempDir()}.Subscribe(t.Context(), []string{"bugs"})
		require.Error(t, err)
	})
}

func TestPollSource(t *testing.T) {
	const period = 20 * time.Millisecond

	// backdate makes the mtimes of the paths polled old enough to be trusted
	backdate := func(t *testing.T, repo *GoGitRepo) {
		t.Helper()
		old := time.Now().Add(-time.Hour)
		for _, path := range []string{
			filepath.Join(repo.path, "packed-refs"),
			filepath.Join(repo.path, "refs", "bugs"),
		} {
			err := os.Chtimes(path, old, old)
			if !errors.Is(err, os.ErrNotExist) {
				require.NoError(t, err)
			}
		}
	}
	subscribe := func(t *testing.T, repo *GoGitRepo) <-chan Change {
		t.Helper()
		ch, err := NewPollSource(repo, period).Subscribe(t.Context(), []string{"bugs", "identities"})
		require.NoError(t, err)
		return ch
	}

	t.Run("nothing changed", func(t *testing.T) {
		repo, _ := newWatchedRepo(t)
		backdate(t, repo)
		ch := subscribe(t, repo)
		requireNoChange(t, ch)
	})

	for _, tc := range []struct {
		name   string
		change func(t *testing.T, repo *GoGitRepo, commits []Hash)
	}{
		{
			name: "a ref written by GoGitRepo",
			change: func(t *testing.T, repo *GoGitRepo, commits []Hash) {
				require.NoError(t, repo.UpdateRef("bugs", randomKey(), "", commits[0]))
			},
		},
		{
			name: "a ref written by the git binary",
			change: func(t *testing.T, repo *GoGitRepo, commits []Hash) {
				requireGitBinary(t)
				runGit(t, repo.path, "update-ref", "refs/bugs/"+randomKey(), commits[1].String())
			},
		},
		{
			name: "a rewrite of packed-refs",
			change: func(t *testing.T, repo *GoGitRepo, commits []Hash) {
				requireGitBinary(t)
				runGit(t, repo.path, "pack-refs", "--all")
			},
		},
		{
			name: "a namespace appearing",
			change: func(t *testing.T, repo *GoGitRepo, commits []Hash) {
				require.NoError(t, repo.UpdateRef("identities", randomKey(), "", commits[0]))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, commits := newWatchedRepo(t)
			backdate(t, repo)
			ch := subscribe(t, repo)
			tc.change(t, repo, commits)
			require.Equal(t, Change{}, nextChange(t, ch))
		})
	}

	t.Run("a recent mtime is reported again", func(t *testing.T) {
		// written within pollGranularity of the poll, a path could change again
		// with the same mtime
		repo, _ := newWatchedRepo(t)
		ch := subscribe(t, repo)
		require.Equal(t, Change{}, nextChange(t, ch))
	})

	t.Run("fails without a refs directory", func(t *testing.T) {
		_, err := pollSource{gitDir: t.TempDir(), period: period}.Subscribe(t.Context(), []string{"bugs"})
		require.Error(t, err)
	})
}

func TestAutoSource(t *testing.T) {
	newAuto := func(watch, poll ChangeSource, canWatch bool) autoSource {
		return autoSource{watch: watch, poll: poll, canWatch: canWatch}
	}

	t.Run("polls where watching can't work", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			watch, poll := newTestSource(), newTestSource()
			auto := newAuto(watch, poll, false)
			ch, err := auto.Subscribe(t.Context(), nil)
			require.NoError(t, err)

			poll.changes <- Change{}
			require.Equal(t, Change{}, <-ch)
		})
	})

	t.Run("watches where it can", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			watch, poll := newTestSource(), newTestSource()
			auto := newAuto(watch, poll, true)
			ch, err := auto.Subscribe(t.Context(), nil)
			require.NoError(t, err)

			watch.changes <- named("bugs", "a")
			require.Equal(t, named("bugs", "a"), <-ch)
		})
	})

	t.Run("polls if the watch can't start", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			poll := newTestSource()
			auto := newAuto(&testSource{err: errors.New("no watch")}, poll, true)
			ch, err := auto.Subscribe(t.Context(), nil)
			require.NoError(t, err)

			poll.changes <- Change{}
			require.Equal(t, Change{}, <-ch)
		})
	})

	t.Run("falls back to polling for good if the watch stops", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			watch, poll := newTestSource(), newTestSource()
			auto := newAuto(watch, poll, true)
			ch, err := auto.Subscribe(t.Context(), nil)
			require.NoError(t, err)

			close(watch.changes)
			// changes may have been missed in between
			require.Equal(t, Change{}, <-ch)

			poll.changes <- named("bugs", "a")
			require.Equal(t, named("bugs", "a"), <-ch)
		})
	})

	t.Run("closes when ctx is done", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			auto := newAuto(newTestSource(), newTestSource(), true)
			ch, err := auto.Subscribe(ctx, nil)
			require.NoError(t, err)
			cancel()
			for range ch {
			}
		})
	})
}
