package cache

import (
	"fmt"
	"io"
	"strings"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util"
)

type RepoCacher interface {
	Repository() repository.Repo
	ResolveBug(id string) (BugCacher, error)
	ResolveBugPrefix(prefix string) (BugCacher, error)
	AllBugIds() ([]string, error)
	ClearAllBugs()

	// Mutations
	NewBug(title string, message string) (BugCacher, error)
	NewBugWithFiles(title string, message string, files []util.Hash) (BugCacher, error)
	Fetch(remote string) (string, error)
	MergeAll(remote string) <-chan bug.MergeResult
	Pull(remote string, out io.Writer) error
	Push(remote string) (string, error)
}

type RepoCache struct {
	repo repository.Repo
	bugs map[string]BugCacher
}

func NewRepoCache(r repository.Repo) RepoCacher {
	return &RepoCache{
		repo: r,
		bugs: make(map[string]BugCacher),
	}
}

func (c *RepoCache) Repository() repository.Repo {
	return c.repo
}

func (c *RepoCache) ResolveBug(id string) (BugCacher, error) {
	cached, ok := c.bugs[id]
	if ok {
		return cached, nil
	}

	b, err := bug.ReadLocalBug(c.repo, id)
	if err != nil {
		return nil, err
	}

	cached = NewBugCache(c.repo, b)
	c.bugs[id] = cached

	return cached, nil
}

func (c *RepoCache) ResolveBugPrefix(prefix string) (BugCacher, error) {
	// preallocate but empty
	matching := make([]string, 0, 5)

	for id := range c.bugs {
		if strings.HasPrefix(id, prefix) {
			matching = append(matching, id)
		}
	}

	// TODO: should check matching bug in the repo as well

	if len(matching) > 1 {
		return nil, fmt.Errorf("Multiple matching bug found:\n%s", strings.Join(matching, "\n"))
	}

	if len(matching) == 1 {
		b := c.bugs[matching[0]]
		return b, nil
	}

	b, err := bug.FindLocalBug(c.repo, prefix)

	if err != nil {
		return nil, err
	}

	cached := NewBugCache(c.repo, b)
	c.bugs[b.Id()] = cached

	return cached, nil
}

func (c *RepoCache) AllBugIds() ([]string, error) {
	return bug.ListLocalIds(c.repo)
}

func (c *RepoCache) ClearAllBugs() {
	c.bugs = make(map[string]BugCacher)
}

func (c *RepoCache) NewBug(title string, message string) (BugCacher, error) {
	return c.NewBugWithFiles(title, message, nil)
}

func (c *RepoCache) NewBugWithFiles(title string, message string, files []util.Hash) (BugCacher, error) {
	author, err := bug.GetUser(c.repo)
	if err != nil {
		return nil, err
	}

	b, err := operations.CreateWithFiles(author, title, message, files)
	if err != nil {
		return nil, err
	}

	err = b.Commit(c.repo)
	if err != nil {
		return nil, err
	}

	cached := NewBugCache(c.repo, b)
	c.bugs[b.Id()] = cached

	return cached, nil
}

func (c *RepoCache) Fetch(remote string) (string, error) {
	return bug.Fetch(c.repo, remote)
}

func (c *RepoCache) MergeAll(remote string) <-chan bug.MergeResult {
	return bug.MergeAll(c.repo, remote)
}

func (c *RepoCache) Pull(remote string, out io.Writer) error {
	return bug.Pull(c.repo, out, remote)
}

func (c *RepoCache) Push(remote string) (string, error) {
	return bug.Push(c.repo, remote)
}
