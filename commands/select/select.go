package _select

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/pkg/errors"
)

const selectFile = "select"

var ErrNoValidId = errors.New("you must provide a bug id")

// ResolveBug first try to resolve a bug using the first argument of the command
// line. If it fails, it fallback to the select mechanism.
//
// Returns:
// - the bug if any
// - the new list of command line arguments with the bug prefix removed if it
//   has been used
// - an error if the process failed
func ResolveBug(repo *cache.RepoCache, args []string) (*cache.BugCache, []string, error) {
	if len(args) > 0 {
		b, err := repo.ResolveBugPrefix(args[0])

		if err == nil {
			return b, args[1:], nil
		}

		if err != bug.ErrBugNotExist {
			return nil, nil, err
		}
	}

	// first arg is not a valid bug prefix

	b, err := selected(repo)
	if err != nil {
		return nil, nil, err
	}

	if b != nil {
		return b, args, nil
	}

	return nil, nil, ErrNoValidId
}

// Select will select a bug for future use
func Select(repo *cache.RepoCache, id string) error {
	selectPath := selectFilePath(repo.Repository())

	f, err := os.OpenFile(selectPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	_, err = f.WriteString(id)
	if err != nil {
		return err
	}

	return f.Close()
}

// Clear will clear the selected bug, if any
func Clear(repo *cache.RepoCache) error {
	selectPath := selectFilePath(repo.Repository())

	return os.Remove(selectPath)
}

func selected(repo *cache.RepoCache) (*cache.BugCache, error) {
	selectPath := selectFilePath(repo.Repository())

	f, err := os.Open(selectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	buf, err := ioutil.ReadAll(io.LimitReader(f, 100))
	if err != nil {
		return nil, err
	}
	if len(buf) == 100 {
		return nil, fmt.Errorf("the select file should be < 100 bytes")
	}

	h := git.Hash(buf)
	if !h.IsValid() {
		err = os.Remove(selectPath)
		if err != nil {
			return nil, errors.Wrap(err, "error while removing invalid select file")
		}

		return nil, fmt.Errorf("select file in invalid, removing it")
	}

	b, err := repo.ResolveBug(string(h))
	if err != nil {
		return nil, err
	}

	err = f.Close()
	if err != nil {
		return nil, err
	}

	return b, nil
}

func selectFilePath(repo repository.Repo) string {
	return path.Join(repo.GetPath(), ".git", "git-bug", selectFile)
}
