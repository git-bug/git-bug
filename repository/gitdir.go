package repository

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// FindGitDir returns the git directory of the repository at path or at one of
// its parents. In each directory, from path up, it looks for:
//   - a .git directory,
//   - a .git file, "gitdir: <path>", as submodules and linked worktrees have,
//   - the directory itself being a git directory, as a bare repository is.
//
// For a linked worktree, it returns the git directory of the main worktree,
// which holds the objects and refs.
//
// It fails with ErrNotARepo if there is none up to the root, and stops with
// an error at a .git file that is malformed or doesn't point to a git
// directory, as git does.
func FindGitDir(path string) (string, error) {
	dir, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	for {
		gitDir, err := gitDirAt(dir)
		if !errors.Is(err, ErrNotARepo) {
			return gitDir, err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%w: %s, or any of its parents", ErrNotARepo, path)
		}
		dir = parent
	}
}

// gitDirAt returns the git directory of the repository at dir, without
// looking at its parents, see FindGitDir.
func gitDirAt(dir string) (string, error) {
	dot := filepath.Join(dir, ".git")
	fi, err := os.Stat(dot)
	switch {
	case err == nil && fi.Mode().IsRegular():
		gitDir, err := readGitFile(dot)
		if err != nil {
			return "", err
		}
		ok, err := isGitDir(gitDir)
		if err != nil {
			return "", err
		}
		if !ok {
			// not ErrNotARepo: the search stops here
			return "", fmt.Errorf("%s points to %s, which is not a git directory", dot, gitDir)
		}
		return gitDir, nil

	case err == nil && fi.IsDir():
		ok, err := isGitDir(dot)
		if err != nil {
			return "", err
		}
		if ok {
			return dot, nil
		}
		// as git, a .git directory that isn't a git directory is ignored

	case err != nil && !isMissing(err):
		return "", err
	}

	ok, err := isGitDir(dir)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrNotARepo
	}
	return dir, nil
}

// readGitFile returns the git directory a .git file at path points to, with
// a "gitdir: <path>" line, relative to the directory of the file or not. For
// a linked worktree, whose git directory has a commondir file, it is the
// common directory that commondir names.
func readGitFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	gitDir, ok := strings.CutPrefix(strings.TrimRight(string(data), "\r\n"), "gitdir: ")
	if !ok || gitDir == "" {
		return "", fmt.Errorf("invalid gitfile format: %s", path)
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(filepath.Dir(path), gitDir)
	}

	commonDir, err := os.ReadFile(filepath.Join(gitDir, "commondir"))
	if isMissing(err) {
		return filepath.Clean(gitDir), nil
	}
	if err != nil {
		return "", err
	}
	common := strings.TrimRight(string(commonDir), "\r\n")
	if !filepath.IsAbs(common) {
		common = filepath.Join(gitDir, common)
	}
	return filepath.Clean(common), nil
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
