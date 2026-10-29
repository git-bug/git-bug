package repository

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
	requireGitBinary(t)
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, "GIT_") {
			t.Setenv(name, "")
			require.NoError(t, os.Unsetenv(name))
		}
	}
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)

	plainRoot := t.TempDir()
	runGit(t, plainRoot, "init")
	plainGitDir, err := filepath.EvalSymlinks(filepath.Join(plainRoot, ".git"))
	require.NoError(t, err)
	bareRoot := t.TempDir()
	runGit(t, bareRoot, "init", "--bare")
	bareGitDir, err := filepath.EvalSymlinks(bareRoot)
	require.NoError(t, err)

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
			{name: "bare", in: bareRoot, expected: bareGitDir},
			{name: "bare subdirectory", in: filepath.Join(bareRoot, "objects"), expected: bareGitDir},
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
		dir := t.TempDir()
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

		sub := filepath.Join(dir, "nested")
		require.NoError(t, os.Mkdir(sub, 0755))
		gitDir, err = FindGitDir(sub)
		require.NoError(t, err)
		require.Equal(t, plainGitDir, gitDir)
	})

	t.Run("a linked worktree gives the common git directory", func(t *testing.T) {
		runGit(t, plainRoot, "commit", "--allow-empty", "-m", "initial")
		dir := filepath.Join(t.TempDir(), "wt")
		runGit(t, plainRoot, "worktree", "add", "--detach", dir)
		sub := filepath.Join(dir, "nested")
		require.NoError(t, os.Mkdir(sub, 0755))
		for _, path := range []string{dir, sub, filepath.Join(plainGitDir, "worktrees", "wt")} {
			gitDir, err := FindGitDir(path)
			require.NoError(t, err)
			require.Equal(t, plainGitDir, gitDir)
		}
	})

	t.Run("submodule through a symlink", func(t *testing.T) {
		root := t.TempDir()
		source := filepath.Join(root, "source")
		super := filepath.Join(root, "real", "super")
		require.NoError(t, os.MkdirAll(source, 0755))
		require.NoError(t, os.MkdirAll(super, 0755))
		runGit(t, source, "init")
		runGit(t, source, "commit", "--allow-empty", "-m", "initial")
		runGit(t, super, "init")
		runGit(t, super, "-c", "protocol.file.allow=always", "submodule", "add", source, "plugins/mod")

		link := filepath.Join(root, "plugins")
		if err := os.Symlink(filepath.Join(super, "plugins"), link); err != nil {
			t.Skipf("cannot create a directory symlink: %v", err)
		}
		logical := filepath.Join(link, "mod")
		sub := filepath.Join(logical, "nested")
		require.NoError(t, os.Mkdir(sub, 0755))
		expected, err := filepath.EvalSymlinks(filepath.Join(super, ".git", "modules", "plugins", "mod"))
		require.NoError(t, err)
		for _, path := range []string{logical, sub} {
			gitDir, err := FindGitDir(path)
			require.NoError(t, err)
			require.Equal(t, expected, gitDir)
		}

		// Match a shell that preserves the logical working directory in PWD.
		t.Chdir(logical)
		t.Setenv("PWD", logical)
		cwd, err := os.Getwd()
		require.NoError(t, err)
		gitDir, err := FindGitDir(cwd)
		require.NoError(t, err)
		require.Equal(t, expected, gitDir)
		gitDir, err = FindGitDir(".")
		require.NoError(t, err)
		require.Equal(t, expected, gitDir)

		// Do not clean away ".." before the OS follows the symlink.
		gitDir, err = FindGitDir(link + string(os.PathSeparator) + "..")
		require.NoError(t, err)
		superGitDir, err := filepath.EvalSymlinks(filepath.Join(super, ".git"))
		require.NoError(t, err)
		require.Equal(t, superGitDir, gitDir)
	})

	t.Run("whitespace in the git directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows paths do not preserve trailing spaces")
		}
		dir := filepath.Join(t.TempDir(), "bare repository ")
		require.NoError(t, os.Mkdir(dir, 0755))
		runGit(t, dir, "init", "--bare")
		expected, err := filepath.EvalSymlinks(dir)
		require.NoError(t, err)
		gitDir, err := FindGitDir(dir)
		require.NoError(t, err)
		require.Equal(t, expected, gitDir)
	})

	t.Run("GIT_DIR overrides discovery", func(t *testing.T) {
		t.Setenv("GIT_DIR", bareGitDir)
		gitDir, err := FindGitDir(plainRoot)
		require.NoError(t, err)
		require.Equal(t, bareGitDir, gitDir)
	})

	t.Run("discovery respects ceiling directories", func(t *testing.T) {
		sub := filepath.Join(plainRoot, "ceiling", "nested")
		require.NoError(t, os.MkdirAll(sub, 0755))
		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(sub))
		_, err := FindGitDir(sub)
		require.ErrorIs(t, err, ErrNotARepo)
	})

	t.Run("a broken .git file stops the search", func(t *testing.T) {
		malformed := filepath.Join(plainRoot, "malformed")
		require.NoError(t, os.Mkdir(malformed, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(malformed, ".git"), []byte("garbage"), 0600))
		_, err := FindGitDir(malformed)
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrNotARepo)
		require.ErrorContains(t, err, "invalid gitfile format")

		dangling := filepath.Join(plainRoot, "dangling")
		require.NoError(t, os.Mkdir(dangling, 0755))
		writeGitFile(t, dangling, t.TempDir())
		_, err = FindGitDir(dangling)
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrNotARepo)
		require.ErrorAs(t, err, new(*exec.ExitError))
	})

	t.Run("a directory that can't be read stops the search", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("needs unix permissions, enforced")
		}
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

	t.Run("git is missing", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		_, err := FindGitDir(plainRoot)
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrNotARepo)
		require.ErrorContains(t, err, "git")
	})

	t.Run("git without path-format support", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("the fake git is a shell script")
		}
		bin := t.TempDir()
		// Old rev-parse echoes an unknown option and then a relative path.
		script := "#!/bin/sh\nprintf '%s\\n' --path-format=absolute .git\n"
		require.NoError(t, os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0755))
		t.Setenv("PATH", bin)
		_, err := FindGitDir(plainRoot)
		require.ErrorContains(t, err, "git 2.31 or later is required")
		require.NotErrorIs(t, err, ErrNotARepo)
	})
}
