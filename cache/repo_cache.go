package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
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

	err := c.lock()
	if err != nil {
		return &RepoCache{}, err
	}

	err = c.loadExcerpts()
	if err == nil {
		return c, nil
	}

	c.buildAllExcerpt()

	return c, c.writeExcerpts()
}

// Repository return the underlying repository.
// If you use this, make sure to never change the repo state.
func (c *RepoCache) Repository() repository.Repo {
	return c.repo
}

func (c *RepoCache) lock() error {
	lockPath := repoLockFilePath(c.repo)

	err := repoIsAvailable(c.repo)
	if err != nil {
		return err
	}

	f, err := os.Create(lockPath)
	if err != nil {
		return err
	}

	pid := fmt.Sprintf("%d", os.Getpid())
	_, err = f.WriteString(pid)
	if err != nil {
		return err
	}

	return f.Close()
}

func (c *RepoCache) Close() error {
	lockPath := repoLockFilePath(c.repo)
	return os.Remove(lockPath)
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

func (c *RepoCache) AllBugOrderById() []string {
	result := make([]string, len(c.excerpts))

	i := 0
	for key := range c.excerpts {
		result[i] = key
		i++
	}

	sort.Strings(result)

	return result
}

func (c *RepoCache) AllBugsOrderByEdit() []string {
	excerpts := make([]BugExcerpt, len(c.excerpts))

	i := 0
	for _, val := range c.excerpts {
		excerpts[i] = val
		i++
	}

	sort.Sort(BugsByEditTime(excerpts))

	result := make([]string, len(excerpts))

	for i, val := range excerpts {
		result[i] = val.Id
	}

	return result
}

func (c *RepoCache) AllBugsOrderByCreation() []string {
	excerpts := make([]BugExcerpt, len(c.excerpts))

	i := 0
	for _, val := range c.excerpts {
		excerpts[i] = val
		i++
	}

	sort.Sort(BugsByCreationTime(excerpts))

	result := make([]string, len(excerpts))

	for i, val := range excerpts {
		result[i] = val.Id
	}

	return result
}

// ClearAllBugs clear all bugs kept in memory
func (c *RepoCache) ClearAllBugs() {
	c.bugs = make(map[string]*BugCache)
}

// NewBug create a new bug
// The new bug is written in the repository (commit)
func (c *RepoCache) NewBug(title string, message string) (*BugCache, error) {
	return c.NewBugWithFiles(title, message, nil)
}

// NewBugWithFiles create a new bug with attached files for the message
// The new bug is written in the repository (commit)
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

// Fetch retrieve update from a remote
// This does not change the local bugs state
func (c *RepoCache) Fetch(remote string) (string, error) {
	return bug.Fetch(c.repo, remote)
}

func (c *RepoCache) MergeAll(remote string) <-chan bug.MergeResult {
	return bug.MergeAll(c.repo, remote)
}

// Pull does a Fetch and merge the updates into the local bug states
func (c *RepoCache) Pull(remote string, out io.Writer) error {
	return bug.Pull(c.repo, out, remote)
}

// Push update a remote with the local changes
func (c *RepoCache) Push(remote string) (string, error) {
	return bug.Push(c.repo, remote)
}

func repoLockFilePath(repo repository.Repo) string {
	return path.Join(repo.GetPath(), ".git", "git-bug", lockfile)
}

// repoIsAvailable check is the given repository is locked by a Cache.
// Note: this is a smart function that will cleanup the lock file if the
// corresponding process is not there anymore.
// If no error is returned, the repo is free to edit.
func repoIsAvailable(repo repository.Repo) error {
	lockPath := repoLockFilePath(repo)

	// Todo: this leave way for a racey access to the repo between the test
	// if the file exist and the actual write. It's probably not a problem in
	// practice because using a repository will be done from user interaction
	// or in a context where a single instance of git-bug is already guaranteed
	// (say, a server with the web UI running). But still, that might be nice to
	// have a mutex or something to guard that.

	// Todo: this will fail if somehow the filesystem is shared with another
	// computer. Should add a configuration that prevent the cleaning of the
	// lock file

	f, err := os.Open(lockPath)

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if err == nil {
		// lock file already exist
		buf, err := ioutil.ReadAll(io.LimitReader(f, 10))
		if err != nil {
			return err
		}
		if len(buf) == 10 {
			return fmt.Errorf("The lock file should be < 10 bytes")
		}

		pid, err := strconv.Atoi(string(buf))
		if err != nil {
			return err
		}

		if util.ProcessIsRunning(pid) {
			return fmt.Errorf("The repository you want to access is already locked by the process pid %d", pid)
		}

		// The lock file is just laying there after a crash, clean it

		fmt.Println("A lock file is present but the corresponding process is not, removing it.")
		err = f.Close()
		if err != nil {
			return err
		}

		os.Remove(lockPath)
		if err != nil {
			return err
		}
	}

	return nil
}
