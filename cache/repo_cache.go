package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util"
)

type RepoCache struct {
	repo     repository.Repo
	excerpts map[string]BugExcerpt
	bugs     map[string]*BugCache
}

func NewRepoCache(r repository.Repo) (*RepoCache, error) {
	c := &RepoCache{
		repo: r,
		bugs: make(map[string]*BugCache),
	}

	err := c.loadExcerpts()

	if err == nil {
		return c, nil
	}

	c.buildAllExcerpt()

	return c, c.writeExcerpts()
}

func (c *RepoCache) Repository() repository.Repo {
	return c.repo
}

// bugUpdated is a callback to trigger when the excerpt of a bug changed,
// that is each time a bug is updated
func (c *RepoCache) bugUpdated(id string) error {
	b, ok := c.bugs[id]
	if !ok {
		panic("missing bug in the cache")
	}

	c.excerpts[id] = NewBugExcerpt(b.bug, b.Snapshot())

	return c.writeExcerpts()
}

// loadExcerpts will try to read from the disk the bug excerpt file
func (c *RepoCache) loadExcerpts() error {
	excerptsPath := repoExcerptsFilePath(c.repo)

	f, err := os.Open(excerptsPath)
	if err != nil {
		return err
	}

	decoder := gob.NewDecoder(f)

	var excerpts map[string]BugExcerpt

	err = decoder.Decode(&excerpts)
	if err != nil {
		return err
	}

	c.excerpts = excerpts
	return nil
}

// writeExcerpts will serialize on disk the BugExcerpt array
func (c *RepoCache) writeExcerpts() error {
	var data bytes.Buffer

	encoder := gob.NewEncoder(&data)

	err := encoder.Encode(c.excerpts)
	if err != nil {
		return err
	}

	excerptsPath := repoExcerptsFilePath(c.repo)

	f, err := os.Create(excerptsPath)
	if err != nil {
		return err
	}

	_, err = f.Write(data.Bytes())
	if err != nil {
		return err
	}

	return f.Close()
}

func repoExcerptsFilePath(repo repository.Repo) string {
	return path.Join(repo.GetPath(), ".git", "git-bug", excerptsFile)
}

func (c *RepoCache) buildAllExcerpt() {
	c.excerpts = make(map[string]BugExcerpt)

	allBugs := bug.ReadAllLocalBugs(c.repo)

	for b := range allBugs {
		snap := b.Bug.Compile()
		c.excerpts[b.Bug.Id()] = NewBugExcerpt(b.Bug, &snap)
	}
}

func (c *RepoCache) ResolveBug(id string) (*BugCache, error) {
	cached, ok := c.bugs[id]
	if ok {
		return cached, nil
	}

	b, err := bug.ReadLocalBug(c.repo, id)
	if err != nil {
		return nil, err
	}

	cached = NewBugCache(c, b)
	c.bugs[id] = cached

	return cached, nil
}

func (c *RepoCache) ResolveBugPrefix(prefix string) (*BugCache, error) {
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

	cached := NewBugCache(c, b)
	c.bugs[b.Id()] = cached

	return cached, nil
}

func (c *RepoCache) AllBugIds() ([]string, error) {
	return bug.ListLocalIds(c.repo)
}

func (c *RepoCache) ClearAllBugs() {
	c.bugs = make(map[string]*BugCache)
}

func (c *RepoCache) NewBug(title string, message string) (*BugCache, error) {
	return c.NewBugWithFiles(title, message, nil)
}

func (c *RepoCache) NewBugWithFiles(title string, message string, files []util.Hash) (*BugCache, error) {
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

	cached := NewBugCache(c, b)
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
