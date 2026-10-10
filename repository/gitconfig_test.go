package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/git-bug/gitconfig"
	"github.com/stretchr/testify/require"
)

func TestGitConfig(t *testing.T) {
	newConfig := func(t *testing.T) *gitConfig {
		gitDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0644))
		return &gitConfig{env: gitconfig.Env{
			GitDir:       gitDir,
			GlobalConfig: []string{filepath.Join(t.TempDir(), ".gitconfig")},
		}}
	}

	t.Run("reads merge the scopes, updates edit the repository's file", func(t *testing.T) {
		cfg := newConfig(t)
		global := "[merged]\n\tboth = global\n\tglobal = global\n"
		require.NoError(t, os.WriteFile(cfg.env.GlobalConfig[0], []byte(global), 0644))

		require.NoError(t, cfg.Update(func(f *gitconfig.File) error {
			return f.Set("merged.both", "local")
		}))

		loaded, err := cfg.Read()
		require.NoError(t, err)
		require.Equal(t, "local", loaded.Value("merged.both", ""))
		require.Equal(t, "global", loaded.Value("merged.global", ""))

		data, err := os.ReadFile(cfg.env.GlobalConfig[0])
		require.NoError(t, err)
		require.Equal(t, global, string(data))
	})

	t.Run("concurrent updates don't lose each other's edits", func(t *testing.T) {
		cfg := newConfig(t)

		const writers = 20
		var wg sync.WaitGroup
		for i := range writers {
			wg.Go(func() {
				require.NoError(t, cfg.Update(func(f *gitconfig.File) error {
					return f.Set(fmt.Sprintf("git-bug.writer%d", i), "value")
				}))
			})
		}
		wg.Wait()

		loaded, err := cfg.Read()
		require.NoError(t, err)
		n := 0
		for range loaded.Section("git-bug") {
			n++
		}
		require.Equal(t, writers, n)
	})

	t.Run("a held lock is waited for", func(t *testing.T) {
		cfg := newConfig(t)
		lock := filepath.Join(cfg.env.GitDir, "config.lock")

		require.NoError(t, os.WriteFile(lock, nil, 0644))
		time.AfterFunc(100*time.Millisecond, func() { _ = os.Remove(lock) })

		require.NoError(t, cfg.Update(func(f *gitconfig.File) error {
			return f.Set("git-bug.key", "value")
		}))
		loaded, err := cfg.Read()
		require.NoError(t, err)
		require.Equal(t, "value", loaded.Value("git-bug.key", ""))
	})

	t.Run("a lock never released is an error", func(t *testing.T) {
		cfg := newConfig(t)
		require.NoError(t, os.WriteFile(filepath.Join(cfg.env.GitDir, "config.lock"), nil, 0644))

		saved := configLockTimeout
		configLockTimeout = 50 * time.Millisecond
		t.Cleanup(func() { configLockTimeout = saved })

		err := cfg.Update(func(f *gitconfig.File) error {
			return f.Set("git-bug.key", "value")
		})
		require.ErrorIs(t, err, gitconfig.ErrLocked)
	})
}
