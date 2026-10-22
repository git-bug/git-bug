package repository

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/git-bug/gitconfig"
	"github.com/stretchr/testify/require"
)

func TestOpenDefaultGitTransport(t *testing.T) {
	// neither the user's configuration nor keyring
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	_, errGit := exec.LookPath("git")

	tests := []struct {
		name      string
		transport string // empty for unset
		expected  RepoRemote
	}{
		{name: "unset", expected: &GoGitRepo{}},
		{name: "go-git", transport: "go-git", expected: &GoGitRepo{}},
		{name: "git", transport: "git", expected: &GitRepo{}},
		{name: "invalid", transport: "invalid", expected: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.transport == "git" && errGit != nil {
				t.Skip("git binary not available")
			}

			dir := t.TempDir()
			_, err := InitGoGitRepo(dir)
			require.NoError(t, err)
			gitDir := filepath.Join(dir, ".git")

			if tt.transport != "" {
				err = NewGitConfig(gitconfig.Env{GitDir: gitDir}).Update(func(f *gitconfig.File) error {
					return f.Set(gitTransportConfigKey, tt.transport)
				})
				require.NoError(t, err)
			}

			repo, err := OpenDefault(gitDir, "git-bug")
			if tt.expected == nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, repo.Close()) })
			require.IsType(t, tt.expected, repo.(*ComposedRepo).RepoRemote)
		})
	}
}
