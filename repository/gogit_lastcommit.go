package repository

import (
	"container/heap"
	"errors"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	commitgraph "github.com/go-git/go-git/v5/plumbing/format/commitgraph/v2"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// lastCommitWalk walks the history from start in committer-time order, and
// finds for each name of pending the most recent commit that changed that entry
// of the directory at path, compared to the commit's first parent. It removes
// the names it finds from pending. The walk stops when pending is empty, or
// after lastCommitDepthLimit commits, after which it finds the names left with
// lastCommitBisect if the repository has a commit-graph.
//
// The trees along path are compared by hash from the root down, and read only
// while they differ: a commit that left the directory alone costs its commit
// and its root tree, and the directory is read only when it changed.
func (repo *GoGitRepo) lastCommitWalk(start plumbing.Hash, path string, pending map[string]bool) (map[string]CommitMeta, error) {
	var components []string
	if path != "" {
		components = strings.Split(path, "/")
	}

	r := newHistoryReader(repo)
	defer r.close()

	first, err := r.commit(start)
	if err != nil || first == nil {
		return nil, err
	}

	result := make(map[string]CommitMeta, len(pending))
	queue := &commitQueue{first}
	seen := map[plumbing.Hash]bool{start: true}

	for depth := 0; len(pending) > 0 && queue.Len() > 0 && depth < lastCommitDepthLimit; depth++ {
		c := heap.Pop(queue).(*walkCommit)

		// A missing parent is treated as no parent.
		parentTree := plumbing.ZeroHash
		for i, h := range c.parents {
			if i > 0 && seen[h] {
				continue
			}
			p, err := r.commit(h)
			if err != nil {
				return nil, err
			}
			if p == nil {
				continue
			}
			if i == 0 {
				parentTree = p.tree
			}
			if !seen[h] {
				seen[h] = true
				heap.Push(queue, p)
			}
		}

		dir, parentDir, err := r.dirs(c.tree, parentTree, components)
		if err != nil {
			return nil, err
		}
		if dir == parentDir {
			continue
		}

		current, err := r.entries(dir)
		if err != nil {
			return nil, err
		}
		parent, err := r.entries(parentDir)
		if err != nil {
			return nil, err
		}
		// A missing entry has the zero hash.
		var changed []string
		for name := range pending {
			if current[name].Hash != parent[name].Hash {
				changed = append(changed, name)
			}
		}
		if len(changed) == 0 {
			continue
		}
		meta, err := r.meta(c)
		if err != nil {
			return nil, err
		}
		for _, name := range changed {
			result[name] = meta
			delete(pending, name)
		}
	}

	if len(pending) > 0 && r.graph != nil && queue.Len() > 0 {
		if err := lastCommitBisect(r, first, components, pending, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// lastCommitBisect finds, for each name of pending, a commit of the
// first-parent chain of start that gave its entry in the directory at the path
// of components its content at start, and adds it to result. It is for the
// names left after lastCommitWalk's depth limit: they are old, and walking the
// chain in the commit-graph reads no tree, so each name is found by galloping
// then bisecting along the chain, reading the trees of a few of its commits
// only.
//
// The result can be wrong. Bisecting assumes that, going back from start, the
// entry keeps its content at start, then differs. An entry changed then
// changed back to that same content, as by git revert, breaks it: if no probe
// falls in between, the search goes past the change back to the content at
// start, and finds an older commit. For example, with the content of the entry
// along the chain:
//
//	index    0  1  2  3  4  5  6  7  8  …  40  41
//	content  A  A  A  A  A  B  A  A  A  …  A   C
//
// the answer is 4, which changed the entry back to A, but the gallop probes 1,
// 3, 7, 15 and 31, all A, missing the B at 5, and the search finds 40. The
// narrower and older the change, the more likely it is missed.
//
// TODO: the walk is exact but reads a root tree per commit, which is why it
// has a depth limit and this approximation exists. Two ways to get rid of
// both:
//   - read the changed-path Bloom filters of the commit-graph, which git
//     writes with git commit-graph write --changed-paths and go-git doesn't
//     read: the walk would then read trees only for the commits that may have
//     changed path, making a walk of the whole history affordable when the
//     filters are there;
//   - keep the results of the walk in the cache, by commit and path, rather
//     than in memory by tree: a walk could then stop at the first commit it
//     already knows, the whole history being walked once, in the background,
//     with the result given as it comes.
func lastCommitBisect(r *historyReader, start *walkCommit, components []string, pending map[string]bool, result map[string]CommitMeta) error {
	// The first-parent chain of start, read as far as needed.
	chain := []*walkCommit{start}
	// reach reads the chain up to i, and reports whether it goes that far.
	reach := func(i int) (bool, error) {
		for len(chain) <= i {
			last := chain[len(chain)-1]
			if len(last.parents) == 0 {
				return false, nil
			}
			p, err := r.commit(last.parents[0])
			if err != nil || p == nil {
				return false, err
			}
			chain = append(chain, p)
		}
		return true, nil
	}
	// content returns the hash of the entry name at the i-th commit of the
	// chain, or the zero hash if there is none.
	content := func(i int, name string) (plumbing.Hash, error) {
		dir, err := r.dir(chain[i].tree, components)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		e, err := r.entries(dir)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		return e[name].Hash, nil
	}

	for name := range pending {
		want, err := content(0, name)
		if err != nil {
			return err
		}
		// The entry has its content at start at lo, and not at hi, where
		// hi == len(chain) stands for beyond the root of the chain. The
		// gallop probes 1, 3, 7, 15…
		lo, hi := 0, 1
		for {
			ok, err := reach(hi)
			if err != nil {
				return err
			}
			if !ok {
				hi = len(chain)
				break
			}
			got, err := content(hi, name)
			if err != nil {
				return err
			}
			if got != want {
				break
			}
			lo, hi = hi, 2*hi+1
		}
		for hi-lo > 1 {
			mid := lo + (hi-lo)/2
			got, err := content(mid, name)
			if err != nil {
				return err
			}
			if got == want {
				lo = mid
			} else {
				hi = mid
			}
		}

		meta, err := r.meta(chain[lo])
		if err != nil {
			return err
		}
		result[name] = meta
		delete(pending, name)
	}
	return nil
}

// historyReader reads the commits and trees of a repository for
// lastCommitWalk. It reads a commit from the commit-graph when it's there,
// without reading the commit itself, and keeps the trees read recently.
//
// It takes rMutex for each read only, so that other requests run during a long
// walk: it reads objects by hash, which never change.
type historyReader struct {
	repo  *GoGitRepo
	graph commitgraph.Index // nil if the repository has none

	// trees holds the entries of the trees read recently, by tree hash. In a
	// linear history, the parent's trees of a step are the commit's trees of
	// the next.
	trees map[plumbing.Hash]map[string]object.TreeEntry
}

// walkCommit is what lastCommitWalk needs of a commit, with the commit itself
// when it was read rather than found in the commit-graph.
type walkCommit struct {
	hash    plumbing.Hash
	tree    plumbing.Hash
	parents []plumbing.Hash
	when    time.Time
	commit  *object.Commit
}

func newHistoryReader(repo *GoGitRepo) *historyReader {
	return &historyReader{
		repo:  repo,
		graph: repo.openCommitGraph(),
		trees: make(map[plumbing.Hash]map[string]object.TreeEntry),
	}
}

func (r *historyReader) close() {
	if r.graph != nil {
		_ = r.graph.Close()
	}
}

// commit returns the commit h, or nil if it's missing from the repository, as
// in a shallow clone.
func (r *historyReader) commit(h plumbing.Hash) (*walkCommit, error) {
	if r.graph != nil {
		if i, err := r.graph.GetIndexByHash(h); err == nil {
			d, err := r.graph.GetCommitDataByIndex(i)
			if err != nil {
				return nil, err
			}
			return &walkCommit{hash: h, tree: d.TreeHash, parents: d.ParentHashes, when: d.When}, nil
		}
	}
	c, err := r.readCommit(h)
	if errors.Is(err, plumbing.ErrObjectNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &walkCommit{hash: h, tree: c.TreeHash, parents: c.ParentHashes, when: c.Committer.When, commit: c}, nil
}

// meta returns the CommitMeta of c, reading the commit if it came from the
// commit-graph.
func (r *historyReader) meta(c *walkCommit) (CommitMeta, error) {
	commit := c.commit
	if commit == nil {
		var err error
		commit, err = r.readCommit(c.hash)
		if err != nil {
			return CommitMeta{}, err
		}
	}
	return commitToMeta(commit), nil
}

// entries returns the entries of the tree h by name, none for the zero hash.
func (r *historyReader) entries(h plumbing.Hash) (map[string]object.TreeEntry, error) {
	if h.IsZero() {
		return nil, nil
	}
	if e, ok := r.trees[h]; ok {
		return e, nil
	}
	tree, err := r.readTree(h)
	if err != nil {
		return nil, err
	}
	e := make(map[string]object.TreeEntry, len(tree.Entries))
	for _, entry := range tree.Entries {
		e[entry.Name] = entry
	}
	if len(r.trees) >= 64 {
		clear(r.trees)
	}
	r.trees[h] = e
	return e, nil
}

// subtree returns the hash of the directory name in the tree h, or the zero
// hash if there is none.
func (r *historyReader) subtree(h plumbing.Hash, name string) (plumbing.Hash, error) {
	e, err := r.entries(h)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	entry, ok := e[name]
	if !ok || entry.Mode != filemode.Dir {
		return plumbing.ZeroHash, nil
	}
	return entry.Hash, nil
}

// dir returns the hash of the directory at the path of components in the root
// tree h, or the zero hash if there is none.
func (r *historyReader) dir(h plumbing.Hash, components []string) (plumbing.Hash, error) {
	for _, name := range components {
		var err error
		if h, err = r.subtree(h, name); err != nil {
			return plumbing.ZeroHash, err
		}
	}
	return h, nil
}

// dirs returns the hashes of the directory at the path of components in the
// root trees a and b. It stops going down as soon as the trees are the same,
// returning then equal hashes for an unchanged directory without reading it.
func (r *historyReader) dirs(a, b plumbing.Hash, components []string) (plumbing.Hash, plumbing.Hash, error) {
	for _, name := range components {
		if a == b {
			break
		}
		var err error
		if a, err = r.subtree(a, name); err != nil {
			return plumbing.ZeroHash, plumbing.ZeroHash, err
		}
		if b, err = r.subtree(b, name); err != nil {
			return plumbing.ZeroHash, plumbing.ZeroHash, err
		}
	}
	return a, b, nil
}

func (r *historyReader) readCommit(h plumbing.Hash) (*object.Commit, error) {
	r.repo.rMutex.Lock()
	defer r.repo.rMutex.Unlock()
	return r.repo.r.CommitObject(h)
}

func (r *historyReader) readTree(h plumbing.Hash) (*object.Tree, error) {
	r.repo.rMutex.Lock()
	defer r.repo.rMutex.Unlock()
	return object.GetTree(r.repo.r.Storer, h)
}

// commitQueue is a heap of commits, the most recently committed first.
type commitQueue []*walkCommit

func (q commitQueue) Len() int           { return len(q) }
func (q commitQueue) Less(i, j int) bool { return q[i].when.After(q[j].when) }
func (q commitQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *commitQueue) Push(x any)        { *q = append(*q, x.(*walkCommit)) }
func (q *commitQueue) Pop() any {
	old := *q
	c := old[len(old)-1]
	*q = old[:len(old)-1]
	return c
}
