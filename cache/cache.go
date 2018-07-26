package cache

import (
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/repository"
)

type Cacher interface {
	RegisterRepository(ref string, repo repository.Repo)
	RegisterDefaultRepository(repo repository.Repo)

	ResolveRepo(ref string) (RepoCacher, error)
	DefaultRepo() (RepoCacher, error)

	// Shortcut to resolve on the default repo for convenience
	DefaultResolveBug(id string) (BugCacher, error)
	DefaultResolveBugPrefix(prefix string) (BugCacher, error)
}

type RepoCacher interface {
	ResolveBug(id string) (BugCacher, error)
	ResolveBugPrefix(prefix string) (BugCacher, error)
	ClearAllBugs()
}

type BugCacher interface {
	Snapshot() bug.Snapshot
	ClearSnapshot()
}

// Cacher ------------------------

type RootCache struct {
	repos map[string]RepoCacher
}

func NewCache() Cacher {
	return &RootCache{
		repos: make(map[string]RepoCacher),
	}
}

func (c *RootCache) RegisterRepository(ref string, repo repository.Repo) {
	c.repos[ref] = NewRepoCache(repo)
}

func (c *RootCache) RegisterDefaultRepository(repo repository.Repo) {
	c.repos[""] = NewRepoCache(repo)
}

func (c *RootCache) DefaultRepo() (RepoCacher, error) {
	if len(c.repos) != 1 {
		return nil, fmt.Errorf("repository is not unique")
	}

	for _, r := range c.repos {
		return r, nil
	}

	panic("unreachable")
}

func (c *RootCache) ResolveRepo(ref string) (RepoCacher, error) {
	r, ok := c.repos[ref]
	if !ok {
		return nil, fmt.Errorf("unknown repo")
	}
	return r, nil
}

func (c *RootCache) DefaultResolveBug(id string) (BugCacher, error) {
	repo, err := c.DefaultRepo()

	if err != nil {
		return nil, err
	}

	return repo.ResolveBug(id)
}

func (c *RootCache) DefaultResolveBugPrefix(prefix string) (BugCacher, error) {
	repo, err := c.DefaultRepo()

	if err != nil {
		return nil, err
	}

	return repo.ResolveBugPrefix(prefix)
}

// Repo ------------------------

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

func (c RepoCache) ResolveBug(id string) (BugCacher, error) {
	cached, ok := c.bugs[id]
	if ok {
		return cached, nil
	}

	b, err := bug.ReadLocalBug(c.repo, id)
	if err != nil {
		return nil, err
	}

	cached = NewBugCache(b)
	c.bugs[id] = cached

	return cached, nil
}

func (c RepoCache) ResolveBugPrefix(prefix string) (BugCacher, error) {
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

	cached := NewBugCache(b)
	c.bugs[b.Id()] = cached

	return cached, nil
}

func (c RepoCache) ClearAllBugs() {
	c.bugs = make(map[string]BugCacher)
}

// Bug ------------------------

type BugCache struct {
	bug  *bug.Bug
	snap *bug.Snapshot
}

func NewBugCache(b *bug.Bug) BugCacher {
	return &BugCache{
		bug: b,
	}
}

func (c BugCache) Snapshot() bug.Snapshot {
	if c.snap == nil {
		snap := c.bug.Compile()
		c.snap = &snap
	}
	return *c.snap
}

func (c BugCache) ClearSnapshot() {
	c.snap = nil
}
