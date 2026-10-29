package repository

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/git-bug/gitconfig"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/require"
)

// newTestGoGitRepo creates a GoGitRepo in a directory removed with the test.
func newTestGoGitRepo(t testing.TB, bare bool) *GoGitRepo {
	t.Helper()

	dir := t.TempDir()
	create, gitDir := InitGoGitRepo, filepath.Join(dir, ".git")
	if bare {
		create, gitDir = InitBareGoGitRepo, dir
	}
	require.NoError(t, create(dir))
	config := NewRepoConfig(NewGitConfig(gitconfig.Env{
		GitDir:       gitDir,
		GlobalConfig: []string{filepath.Join(t.TempDir(), ".gitconfig")},
	}))
	repo, err := OpenGoGitRepo(gitDir, config)
	require.NoError(t, err)
	return repo
}

// requireGitBinary skips the test when the git binary is not available.
func requireGitBinary(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
}

// runGit runs the git binary in dir, which can be a worktree or a git
// directory, and returns its trimmed output.
//
// git is isolated from the environment running the tests: no inherited GIT_*
// variable redirecting it (a GIT_DIR set by a hook would win over -C), no
// system or user config, which could run hooks (core.hooksPath) or change its
// behavior, and no automatic gc or maintenance changing the object store
// behind the test's back. Commits get a fixed identity.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	home := t.TempDir()
	env := []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + home,
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "GIT_") && !strings.HasPrefix(kv, "HOME=") && !strings.HasPrefix(kv, "XDG_CONFIG_HOME=") {
			env = append(env, kv)
		}
	}

	args = append([]string{"-C", dir, "-c", "gc.auto=0", "-c", "maintenance.auto=false"}, args...)
	cmd := exec.Command("git", args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s: %s", strings.Join(args, " "), out)
	return strings.TrimSpace(string(out))
}

func TestFindGitDir(t *testing.T) {
	// FindGitDir returns physical paths, and temporary directories can be
	// behind a symbolic link, as on macOS
	physical := func(t *testing.T, path string) string {
		t.Helper()
		path, err := filepath.EvalSymlinks(path)
		require.NoError(t, err)
		return path
	}

	plainGitDir := physical(t, newTestGoGitRepo(t, false).path)
	plainRoot := filepath.Dir(plainGitDir)
	bareGitDir := physical(t, newTestGoGitRepo(t, true).path)

	writeGitFile := func(t *testing.T, dir string, gitDir string) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: "+gitDir+"\n"), 0600))
	}

	t.Run("found from inside", func(t *testing.T) {
		sub := filepath.Join(plainRoot, "a", "b")
		require.NoError(t, os.MkdirAll(sub, 0755))

		tests := []struct {
			name     string
			in       string
			expected string
		}{
			{name: "worktree", in: plainRoot, expected: plainGitDir},
			{name: "worktree subdirectory", in: sub, expected: plainGitDir},
			{name: "git directory", in: plainGitDir, expected: plainGitDir},
			{name: "git directory subdirectory", in: filepath.Join(plainGitDir, "objects"), expected: plainGitDir},
			{name: "bare", in: bareGitDir, expected: bareGitDir},
			{name: "bare subdirectory", in: filepath.Join(bareGitDir, "objects"), expected: bareGitDir},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				gitDir, err := FindGitDir(tt.in)
				require.NoError(t, err)
				require.Equal(t, tt.expected, gitDir)
			})
		}
	})

	t.Run("none", func(t *testing.T) {
		_, err := FindGitDir(t.TempDir())
		require.ErrorIs(t, err, ErrNotARepo)
	})

	t.Run("a .git file, absolute or relative", func(t *testing.T) {
		dir := physical(t, t.TempDir())
		writeGitFile(t, dir, plainGitDir)
		gitDir, err := FindGitDir(dir)
		require.NoError(t, err)
		require.Equal(t, plainGitDir, gitDir)

		rel, err := filepath.Rel(dir, plainGitDir)
		require.NoError(t, err)
		writeGitFile(t, dir, rel)
		gitDir, err = FindGitDir(dir)
		require.NoError(t, err)
		require.Equal(t, plainGitDir, gitDir)
	})

	t.Run("a .git file of a linked worktree gives the main git directory", func(t *testing.T) {
		// git worktree add makes .git/worktrees/<name>, with its own HEAD and a
		// commondir file naming the main git directory, which holds the objects
		// and refs
		wtGitDir := filepath.Join(plainGitDir, "worktrees", "wt")
		require.NoError(t, os.MkdirAll(wtGitDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(wtGitDir, "HEAD"), []byte("ref: refs/heads/wt\n"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(wtGitDir, "commondir"), []byte("../..\n"), 0644))

		dir := t.TempDir()
		writeGitFile(t, dir, wtGitDir)
		gitDir, err := FindGitDir(dir)
		require.NoError(t, err)
		require.Equal(t, plainGitDir, gitDir)
	})

	t.Run("a relative .git file through a symbolic link", func(t *testing.T) {
		// as a submodule reached through a link to one of its parents: the
		// .git file is relative to the physical directory, not to the link
		sub := filepath.Join(plainRoot, "real", "sub")
		require.NoError(t, os.MkdirAll(sub, 0755))
		writeGitFile(t, sub, filepath.Join("..", "..", ".git"))

		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(filepath.Join(plainRoot, "real"), link); err != nil {
			t.Skipf("can't create a symbolic link: %v", err)
		}

		gitDir, err := FindGitDir(filepath.Join(link, "sub"))
		require.NoError(t, err)
		require.Equal(t, plainGitDir, gitDir)
	})

	t.Run("a broken .git file stops the search", func(t *testing.T) {
		// inside a repository, which isn't found instead
		malformed := filepath.Join(plainRoot, "malformed")
		require.NoError(t, os.Mkdir(malformed, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(malformed, ".git"), []byte("garbage"), 0600))
		_, err := FindGitDir(malformed)
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrNotARepo)

		dangling := filepath.Join(plainRoot, "dangling")
		require.NoError(t, os.Mkdir(dangling, 0755))
		writeGitFile(t, dangling, t.TempDir())
		_, err = FindGitDir(dangling)
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrNotARepo)
	})

	t.Run("a directory that can't be read stops the search", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("needs unix permissions, enforced")
		}
		// inside a repository, which isn't found instead
		locked := filepath.Join(plainRoot, "locked")
		require.NoError(t, os.Mkdir(locked, 0000))
		t.Cleanup(func() { _ = os.Chmod(locked, 0755) })

		_, err := FindGitDir(locked)
		require.ErrorIs(t, err, fs.ErrPermission)
	})

	t.Run("a .git directory that isn't a git directory is skipped", func(t *testing.T) {
		dir := filepath.Join(plainRoot, "fake")
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0755))
		gitDir, err := FindGitDir(dir)
		require.NoError(t, err)
		require.Equal(t, plainGitDir, gitDir)
	})
}

// TestGoGitRepo_Specifics tests what the shared suite, run by TestGoGitRepo,
// doesn't.
func TestGoGitRepo_Specifics(t *testing.T) {
	t.Run("OpenGoGitRepo doesn't search", func(t *testing.T) {
		opened := newTestGoGitRepo(t, false)
		gitDir := opened.path

		_, err := OpenGoGitRepo(filepath.Dir(gitDir), opened.config)
		require.ErrorIs(t, err, ErrNotARepo)

		repo, err := OpenGoGitRepo(gitDir, opened.config)
		require.NoError(t, err)
		require.Equal(t, gitDir, repo.path)
	})

	t.Run("StoreTree sorts as git", func(t *testing.T) {
		requireGitBinary(t)

		repo := newTestGoGitRepo(t, false)
		blob, err := repo.StoreData(t.Context(), []byte("content"))
		require.NoError(t, err)
		sub, err := repo.StoreTree(t.Context(), []TreeEntry{{ObjectType: Blob, Hash: blob, Name: "x"}})
		require.NoError(t, err)

		// git compares a directory's name as if it ended with a slash: "a" goes
		// after "a-c" and "a.b", not before
		tree, err := repo.StoreTree(t.Context(), []TreeEntry{
			{ObjectType: Tree, Hash: sub, Name: "a"},
			{ObjectType: Blob, Hash: blob, Name: "a.b"},
			{ObjectType: Blob, Hash: blob, Name: "a-c"},
		})
		require.NoError(t, err)

		// git fsck rejects a tree out of git's order
		commit, err := repo.StoreCommit(t.Context(), tree)
		require.NoError(t, err)
		dir := filepath.Dir(repo.path)
		runGit(t, dir, "update-ref", "refs/test/tree", string(commit))
		runGit(t, dir, "fsck", "--strict")
	})

	t.Run("Head", func(t *testing.T) {
		repo := newTestGoGitRepo(t, false)

		// a new repository's HEAD points to refs/heads/master, which doesn't exist yet
		_, err := repo.Head(t.Context())
		require.ErrorIs(t, err, ErrNotFound)

		commit := storeTestCommits(t, repo, 1)[0]
		require.NoError(t, repo.r.Storer.SetReference(plumbing.NewHashReference(
			plumbing.NewBranchReferenceName("master"), plumbing.NewHash(commit.String()))))

		meta, err := repo.Head(t.Context())
		require.NoError(t, err)
		require.Equal(t, RefMeta{
			Name:      "refs/heads/master",
			ShortName: "master",
			Type:      GitRefTypeBranch,
			Hash:      string(commit),
		}, meta)
	})
}
