package repository

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/execabs"
)

// FindGitDir asks git for the absolute common directory of the repository at
// path. Git handles parent discovery, bare repositories, submodules, linked
// worktrees and the environment variables controlling repository discovery.
// For a linked worktree, the common directory holds the shared objects and refs.
//
// It requires git 2.31 or later in PATH, for rev-parse's --path-format option.
// It fails with ErrNotARepo if git cannot find a repository, and preserves
// other errors, such as a malformed .git file or an unreadable directory.
func FindGitDir(path string) (string, error) {
	cmd := execabs.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	// Let the OS resolve path, including symlinks followed by "..". Cleaning
	// it with filepath.Abs or filepath.Join first can select another directory.
	cmd.Dir = path
	// Git has no distinct exit status for a missing repository. Keep its
	// diagnostic in English so we can recognise that case without hiding
	// malformed gitfiles, permission errors or dubious ownership warnings.
	cmd.Env = append(cmd.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			message := strings.TrimSuffix(string(exitErr.Stderr), "\n")
			if strings.HasPrefix(message, "fatal: not a git repository (or any") {
				return "", fmt.Errorf("%w: git rev-parse: %w: %s", ErrNotARepo, err, message)
			}
			return "", fmt.Errorf("git rev-parse: %w: %s", err, message)
		}
		return "", fmt.Errorf("git rev-parse: %w", err)
	}

	// Strip only the record separator, not whitespace belonging to the path.
	gitDir := strings.TrimSuffix(string(out), "\n")
	if !filepath.IsAbs(gitDir) {
		// Older git versions echo unknown options instead of rejecting them.
		return "", fmt.Errorf("git rev-parse returned a non-absolute path %q; git 2.31 or later is required", gitDir)
	}
	return filepath.FromSlash(gitDir), nil
}

// isGitDir tells if gitDir is a git directory as git recognizes one: it holds
// a HEAD file, or a symbolic link to a branch, and objects and refs
// directories. A path that can't be checked, other than a missing one, is an
// error.
func isGitDir(gitDir string) (bool, error) {
	head := filepath.Join(gitDir, "HEAD")
	fi, err := os.Lstat(head)
	if isMissing(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	switch {
	case fi.Mode().IsRegular():
	case fi.Mode()&fs.ModeSymlink != 0:
		target, err := os.Readlink(head)
		if err != nil {
			return false, err
		}
		if !strings.HasPrefix(filepath.ToSlash(target), "refs/") {
			return false, nil
		}
	default:
		return false, nil
	}

	for _, name := range []string{"objects", "refs"} {
		fi, err := os.Stat(filepath.Join(gitDir, name))
		if isMissing(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !fi.IsDir() {
			return false, nil
		}
	}
	return true, nil
}

// isMissing tells if err means that a path doesn't exist: either it doesn't,
// or one of its parents is not a directory.
func isMissing(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR)
}
