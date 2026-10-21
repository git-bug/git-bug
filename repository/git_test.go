package repository_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/git-bug/gitconfig"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/repository/repotest"
)

// TestGitRepoSSH checks that ssh runs in batch mode, unless the user chose the
// ssh command.
func TestGitRepoSSH(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	if runtime.GOOS == "windows" {
		t.Skip("the fake ssh is a shell script")
	}

	// an ssh that records its arguments in $ARGS_FILE, and fails
	bin := t.TempDir()
	script := "#!/bin/sh\necho \"$@\" > \"$ARGS_FILE\"\nexit 1\n"
	require.NoError(t, os.WriteFile(filepath.Join(bin, "ssh"), []byte(script), 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	// set but empty, git would run an empty command: unset them, Setenv
	// restoring them after the test
	for _, name := range []string{"GIT_SSH_COMMAND", "GIT_SSH"} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}

	tests := []struct {
		name       string
		sshCommand string
		expected   string
	}{
		{name: "default ssh", expected: "-o BatchMode=yes"},
		{name: "user's ssh command", sshCommand: "ssh -o Custom=yes", expected: "-o Custom=yes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsFile := filepath.Join(t.TempDir(), "args")
			t.Setenv("ARGS_FILE", argsFile)

			repo := repotest.NewGoGitRepo(t, false)
			err := repo.Config().Update(func(f *gitconfig.File) error {
				if tt.sshCommand != "" {
					if err := f.Set("core.sshCommand", tt.sshCommand); err != nil {
						return err
					}
				}
				return f.Set("remote.origin.url", "ssh://example.com/repo.git")
			})
			require.NoError(t, err)

			remote, err := repository.NewGitRepo(repo.GitDir, repo.RepoConfig)
			require.NoError(t, err)
			_, err = remote.FetchRefs(t.Context(), "origin", "bugs")
			require.Error(t, err)

			args, err := os.ReadFile(argsFile)
			require.NoError(t, err)
			require.Contains(t, string(args), tt.expected)
			if tt.sshCommand != "" {
				require.NotContains(t, string(args), "BatchMode")
			}
		})
	}
}

// TestGitRepoNoAskpass checks that git doesn't run an askpass program for the
// credentials of an https remote.
func TestGitRepoNoAskpass(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	if runtime.GOOS == "windows" {
		t.Skip("the fake askpass is a shell script")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	// an askpass that records it ran, in $CALLED_FILE
	bin := t.TempDir()
	askpass := filepath.Join(bin, "askpass")
	script := "#!/bin/sh\ntouch \"$CALLED_FILE\"\necho secret\n"
	require.NoError(t, os.WriteFile(askpass, []byte(script), 0o755))
	calledFile := filepath.Join(t.TempDir(), "called")
	t.Setenv("CALLED_FILE", calledFile)
	t.Setenv("GIT_ASKPASS", askpass)
	t.Setenv("SSH_ASKPASS", askpass)
	// no credential helper from the user's configuration
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)

	repo := repotest.NewGoGitRepo(t, false)
	err := repo.Config().Update(func(f *gitconfig.File) error {
		if err := f.Set("core.askPass", askpass); err != nil {
			return err
		}
		return f.Set("remote.origin.url", server.URL+"/repo.git")
	})
	require.NoError(t, err)

	remote, err := repository.NewGitRepo(repo.GitDir, repo.RepoConfig)
	require.NoError(t, err)
	_, err = remote.FetchRefs(t.Context(), "origin", "bugs")
	require.Error(t, err)
	require.NoFileExists(t, calledFile)
}

// TestGitRepoRelativeRemote checks that, as with git, a relative remote URL is
// relative to the worktree, or the git directory of a bare repository, not to
// the current directory, including with a relative git directory.
func TestGitRepoRelativeRemote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}

	tests := []struct {
		name string
		bare bool
		// worktree sets core.worktree, relative to the git directory
		worktree string
		// relativeGitDir gives the git directory relative to the current one
		relativeGitDir bool
	}{
		{name: "worktree"},
		{name: "bare", bare: true},
		{name: "core.worktree", worktree: filepath.Join("..", "elsewhere")},
		{name: "relative git directory", relativeGitDir: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := repotest.NewGoGitRepo(t, true)
			repo := repotest.NewGoGitRepo(t, tt.bare)

			root := filepath.Dir(repo.GitDir)
			switch {
			case tt.bare:
				root = repo.GitDir
			case tt.worktree != "":
				root = filepath.Join(repo.GitDir, tt.worktree)
			}

			url, err := filepath.Rel(root, upstream.GitDir)
			require.NoError(t, err)
			err = repo.Config().Update(func(f *gitconfig.File) error {
				if tt.worktree != "" {
					if err := f.Set("core.worktree", tt.worktree); err != nil {
						return err
					}
				}
				return f.Set("remote.origin.url", url)
			})
			require.NoError(t, err)

			// somewhere else than the root
			cwd := filepath.Join(root, "a", "b")
			gitDir := repo.GitDir
			if tt.relativeGitDir {
				cwd = filepath.Dir(root)
				gitDir, err = filepath.Rel(cwd, repo.GitDir)
				require.NoError(t, err)
			}
			require.NoError(t, os.MkdirAll(cwd, 0o755))
			t.Chdir(cwd)

			remote, err := repository.NewGitRepo(gitDir, repo.RepoConfig)
			require.NoError(t, err)
			_, err = remote.FetchRefs(t.Context(), "origin", "bugs")
			require.NoError(t, err)
		})
	}
}

// TestGitRepoFetchLocalState checks that fetching changes nothing locally but
// the tracking refs of the fetched namespaces, and brings all the data they
// point to, whatever the repository's configuration.
func TestGitRepoFetchLocalState(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}

	// without the system and user config, which could sign the commits or run
	// hooks
	git := func(t *testing.T, gitDir string, args ...string) string {
		t.Helper()
		args = append([]string{"--git-dir=" + gitDir, "-c", "user.name=a", "-c", "user.email=a@a"}, args...)
		cmd := exec.Command("git", args...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return string(out)
	}

	tests := []struct {
		name string
		// config is set on the repository fetching
		config map[string]string
		// env is set for the fetch only
		env map[string]string
		// prepare runs on the repository fetching before the fetch, and
		// returns what check compares to
		prepare func(t *testing.T, gitDir string) string
		check   func(t *testing.T, gitDir string, before string)
	}{
		{
			// a local v1 conflicts with the upstream one, v2 is upstream only
			name:   "tags",
			config: map[string]string{"remote.origin.tagOpt": "--tags"},
			prepare: func(t *testing.T, gitDir string) string {
				git(t, gitDir, "commit", "--allow-empty", "-m", "b")
				git(t, gitDir, "tag", "v1")
				return git(t, gitDir, "for-each-ref", "refs/tags")
			},
			check: func(t *testing.T, gitDir string, before string) {
				require.Equal(t, before, git(t, gitDir, "for-each-ref", "refs/tags"))
			},
		},
		{
			name: "FETCH_HEAD",
			prepare: func(t *testing.T, gitDir string) string {
				git(t, gitDir, "fetch", "origin", "main")
				return git(t, gitDir, "rev-parse", "FETCH_HEAD")
			},
			check: func(t *testing.T, gitDir string, before string) {
				require.Equal(t, before, git(t, gitDir, "rev-parse", "FETCH_HEAD"))
			},
		},
		{
			name: "partial clone",
			config: map[string]string{
				"remote.origin.promisor":           "true",
				"remote.origin.partialclonefilter": "blob:none",
			},
			check: func(t *testing.T, gitDir string, _ string) {
				objects := git(t, gitDir, "rev-list", "--objects", "--missing=print", "refs/remotes/origin/bugs/x")
				require.NotContains(t, objects, "?")
			},
		},
		{
			// as in a hook of another repository
			name: "repository environment",
			env: map[string]string{
				"GIT_DIR":              "elsewhere",
				"GIT_OBJECT_DIRECTORY": "elsewhere/objects",
			},
			check: func(t *testing.T, gitDir string, _ string) {
				objects := git(t, gitDir, "rev-list", "--objects", "--missing=print", "refs/remotes/origin/bugs/x")
				require.NotContains(t, objects, "?")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// upstream has a branch, the tags v1 and v2 and a bug, whose
			// commit adds a file
			upstream := repotest.NewGoGitRepo(t, true)
			git(t, upstream.GitDir, "config", "uploadpack.allowFilter", "true")
			writer := repotest.NewGoGitRepo(t, false)
			worktree := filepath.Dir(writer.GitDir)
			git(t, writer.GitDir, "commit", "--allow-empty", "-m", "a")
			git(t, writer.GitDir, "tag", "v1")
			git(t, writer.GitDir, "tag", "v2")
			require.NoError(t, os.WriteFile(filepath.Join(worktree, "ops"), []byte("operation"), 0o644))
			git(t, writer.GitDir, "--work-tree="+worktree, "add", "ops")
			git(t, writer.GitDir, "commit", "-m", "op")
			git(t, writer.GitDir, "push", upstream.GitDir, "HEAD~1:refs/heads/main", "HEAD:refs/bugs/x", "refs/tags/*:refs/tags/*")

			repo := repotest.NewGoGitRepo(t, false)
			err := repo.Config().Update(func(f *gitconfig.File) error {
				if err := f.Set("remote.origin.url", upstream.GitDir); err != nil {
					return err
				}
				for key, value := range tt.config {
					if err := f.Set(key, value); err != nil {
						return err
					}
				}
				return nil
			})
			require.NoError(t, err)

			var before string
			if tt.prepare != nil {
				before = tt.prepare(t, repo.GitDir)
			}

			for name, value := range tt.env {
				t.Setenv(name, value)
			}
			remote, err := repository.NewGitRepo(repo.GitDir, repo.RepoConfig)
			require.NoError(t, err)
			_, err = remote.FetchRefs(t.Context(), "origin", "bugs")
			require.NoError(t, err)
			for name := range tt.env {
				require.NoError(t, os.Unsetenv(name))
			}

			require.NotEmpty(t, git(t, repo.GitDir, "for-each-ref", "refs/remotes/origin/bugs"))
			tt.check(t, repo.GitDir, before)
		})
	}
}

// TestGitRepoNoHooks checks that fetching and pushing run none of the
// repository's hooks.
func TestGitRepoNoHooks(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	if runtime.GOOS == "windows" {
		t.Skip("the hooks are shell scripts")
	}

	upstream := repotest.NewGoGitRepo(t, true)
	repo := repotest.NewGoGitRepo(t, false)

	// without the system and user config, which could sign the commits or run
	// hooks
	git := func(args ...string) {
		t.Helper()
		args = append([]string{"--git-dir=" + repo.GitDir, "-c", "user.name=a", "-c", "user.email=a@a"}, args...)
		cmd := exec.Command("git", args...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}

	err := repo.Config().Update(func(f *gitconfig.File) error {
		return f.Set("remote.origin.url", upstream.GitDir)
	})
	require.NoError(t, err)

	git("commit", "--allow-empty", "-m", "a")
	git("update-ref", "refs/bugs/x", "HEAD")

	// hooks that record they ran, in $CALLED_FILE, and fail
	calledFile := filepath.Join(t.TempDir(), "called")
	t.Setenv("CALLED_FILE", calledFile)
	script := "#!/bin/sh\ntouch \"$CALLED_FILE\"\nexit 1\n"
	hooks := filepath.Join(repo.GitDir, "hooks")
	require.NoError(t, os.MkdirAll(hooks, 0o755))
	for _, hook := range []string{"pre-push", "reference-transaction"} {
		require.NoError(t, os.WriteFile(filepath.Join(hooks, hook), []byte(script), 0o755))
	}

	remote, err := repository.NewGitRepo(repo.GitDir, repo.RepoConfig)
	require.NoError(t, err)
	_, err = remote.PushRefs(t.Context(), "origin", "bugs")
	require.NoError(t, err)
	_, err = remote.FetchRefs(t.Context(), "origin", "bugs")
	require.NoError(t, err)
	require.NoFileExists(t, calledFile)
}

// TestGitRepoPushConfig checks that pushing sends the refs of the pushed
// namespaces and nothing else, whatever the repository's configuration.
func TestGitRepoPushConfig(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}

	// without the system and user config, which could sign the commits or run
	// hooks
	git := func(t *testing.T, gitDir string, args ...string) string {
		t.Helper()
		args = append([]string{"--git-dir=" + gitDir, "-c", "user.name=a", "-c", "user.email=a@a"}, args...)
		cmd := exec.Command("git", args...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return string(out)
	}

	tests := []struct {
		name string
		// config is set on the repository pushing
		config map[string]string
	}{
		// with an annotated tag pointing to a commit the remote has
		{name: "push.followTags", config: map[string]string{"push.followTags": "true"}},
		// which would push the submodules only
		{name: "push.recurseSubmodules", config: map[string]string{"push.recurseSubmodules": "only"}},
		// the remote supports neither signed pushes nor push options
		{name: "push.gpgSign", config: map[string]string{"push.gpgSign": "true"}},
		{name: "push.pushOption", config: map[string]string{"push.pushOption": "ci.skip"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := repotest.NewGoGitRepo(t, true)
			repo := repotest.NewGoGitRepo(t, false)

			// the remote has the code, but not its tag v1, and the bug is on
			// its own history
			git(t, repo.GitDir, "commit", "--allow-empty", "-m", "code")
			git(t, repo.GitDir, "tag", "-a", "v1", "-m", "v1")
			git(t, repo.GitDir, "push", upstream.GitDir, "HEAD:refs/heads/main")
			git(t, repo.GitDir, "checkout", "--orphan", "bug")
			git(t, repo.GitDir, "commit", "--allow-empty", "-m", "op")
			git(t, repo.GitDir, "update-ref", "refs/bugs/x", "HEAD")

			err := repo.Config().Update(func(f *gitconfig.File) error {
				if err := f.Set("remote.origin.url", upstream.GitDir); err != nil {
					return err
				}
				for key, value := range tt.config {
					if err := f.Set(key, value); err != nil {
						return err
					}
				}
				return nil
			})
			require.NoError(t, err)

			remote, err := repository.NewGitRepo(repo.GitDir, repo.RepoConfig)
			require.NoError(t, err)
			_, err = remote.PushRefs(t.Context(), "origin", "bugs")
			require.NoError(t, err)

			require.NotEmpty(t, git(t, upstream.GitDir, "for-each-ref", "refs/bugs"))
			require.Empty(t, git(t, upstream.GitDir, "for-each-ref", "refs/tags"))
		})
	}
}
