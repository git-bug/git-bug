package cache

import (
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/repository"
)

type Cache interface {
	RegisterRepository(ref string, repo repository.Repo)
	RegisterDefaultRepository(repo repository.Repo)
	ResolveRepo(ref string) (CachedRepo, error)
	DefaultRepo() (CachedRepo, error)
}

type CachedRepo interface {
	ResolveBug(id string) (CachedBug, error)
	ResolveBugPrefix(prefix string) (CachedBug, error)
	ClearAllBugs()
}

type CachedBug interface {
	Snapshot() bug.Snapshot
	ClearSnapshot()
}

// Cache ------------------------

type DefaultCache struct {
	repos map[string]CachedRepo
}

func NewDefaultCache() Cache {
	return &DefaultCache{
		repos: make(map[string]CachedRepo),
	}
}

func (c *DefaultCache) RegisterRepository(ref string, repo repository.Repo) {
	c.repos[ref] = NewCachedRepo(repo)
}

func (c *DefaultCache) RegisterDefaultRepository(repo repository.Repo) {
	c.repos[""] = NewCachedRepo(repo)
}

func (c *DefaultCache) DefaultRepo() (CachedRepo, error) {
	if len(c.repos) != 1 {
		return nil, fmt.Errorf("repository is not unique")
	}

	for _, r := range c.repos {
		return r, nil
	}

	panic("unreachable")
}

func (c *DefaultCache) ResolveRepo(ref string) (CachedRepo, error) {
	r, ok := c.repos[ref]
	if !ok {
		return nil, fmt.Errorf("unknown repo")
	}
	return r, nil
}

// Repo ------------------------

type CachedRepoImpl struct {
	repo repository.Repo
	bugs map[string]CachedBug
}

func NewCachedRepo(r repository.Repo) CachedRepo {
	return &CachedRepoImpl{
		repo: r,
		bugs: make(map[string]CachedBug),
	}
}

func (c CachedRepoImpl) ResolveBug(id string) (CachedBug, error) {
	cached, ok := c.bugs[id]
	if ok {
		return cached, nil
	}

	b, err := bug.ReadLocalBug(c.repo, id)
	if err != nil {
		return nil, err
	}

	cached = NewCachedBug(b)
	c.bugs[id] = cached

	return cached, nil
}

func (c CachedRepoImpl) ResolveBugPrefix(prefix string) (CachedBug, error) {
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

	cached := NewCachedBug(b)
	c.bugs[b.Id()] = cached

	return cached, nil
}

func (c CachedRepoImpl) ClearAllBugs() {
	c.bugs = make(map[string]CachedBug)
}

// Bug ------------------------

type CachedBugImpl struct {
	bug  *bug.Bug
	snap *bug.Snapshot
}

func NewCachedBug(b *bug.Bug) CachedBug {
	return &CachedBugImpl{
		bug: b,
	}
}

func (c CachedBugImpl) Snapshot() bug.Snapshot {
	if c.snap == nil {
		snap := c.bug.Compile()
		c.snap = &snap
	}
	return *c.snap
}

func (c CachedBugImpl) ClearSnapshot() {
	c.snap = nil
}
