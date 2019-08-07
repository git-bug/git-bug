package _select

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

const selectFile = "select"

var ErrNoValidId = errors.New("you must provide a bug id or use the \"select\" command first")

// ResolveBug first try to resolve a bug using the first argument of the command
// line. If it fails, it fallback to the select mechanism.
//
// Returns:
// - the bug if any
// - the new list of command line arguments with the bug prefix removed if it
//   has been used
// - an error if the process failed
func ResolveBug(repo *cache.RepoCache, args []string) (*cache.BugCache, []string, error) {
	// At first, try to use the first argument as a bug prefix
	if len(args) > 0 {
		b, err := repo.ResolveBugPrefix(args[0])

		if err == nil {
			return b, args[1:], nil
		}

		if err != bug.ErrBugNotExist {
			return nil, nil, err
		}
	}

	// first arg is not a valid bug prefix, we can safely use the preselected bug if any

	b, err := selected(repo)

	// selected bug is invalid
	if err == bug.ErrBugNotExist {
		// we clear the selected bug
		err = Clear(repo)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, ErrNoValidId
	}

	// another error when reading the bug
	if err != nil {
		return nil, nil, err
	}

	// bug is successfully retrieved
	if b != nil {
		return b, args, nil
	}

	// no selected bug and no valid first argument
	return nil, nil, ErrNoValidId
}

// Select will select a bug for future use
func Select(repo *cache.RepoCache, id string) error {
	selectPath := selectFilePath(repo)

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
	selectPath := selectFilePath(repo)

	return os.Remove(selectPath)
}

func selected(repo *cache.RepoCache) (*cache.BugCache, error) {
	selectPath := selectFilePath(repo)

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

	id := string(buf)
	if !bug.IDIsValid(id) {
		err = os.Remove(selectPath)
		if err != nil {
			return nil, errors.Wrap(err, "error while removing invalid select file")
		}

		return nil, fmt.Errorf("select file in invalid, removing it")
	}

	b, err := repo.ResolveBug(id)
	if err != nil {
		return nil, err
	}

	err = f.Close()
	if err != nil {
		return nil, err
	}

	return b, nil
}

func selectFilePath(repo repository.RepoCommon) string {
	return path.Join(repo.GetPath(), ".git", "git-bug", selectFile)
}
