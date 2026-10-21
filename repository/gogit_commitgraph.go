package repository

import (
	"bytes"
	"os"
	"path/filepath"

	commitgraph "github.com/go-git/go-git/v5/plumbing/format/commitgraph/v2"
)

// commitGraphMemoryLimit is the size up to which openCommitGraph reads the
// commit-graph in memory. go-git reads a commit of the commit-graph with
// several reads of a few bytes, each a system call when served by the file.
// 32 MiB holds the commit-graph of about half a million commits.
const commitGraphMemoryLimit = 32 << 20

// openCommitGraph returns the commit-graph of the repository, a single file or
// a chain of files, or nil if there is none or it can't be read. The files are
// read in memory if together they fit in commitGraphMemoryLimit, and read as
// needed otherwise. The caller must close the index.
//
// TODO: with go-git v6, open the files with go-billy's mmap instead, as git
// does: commitgraph.OpenChainOrFileIndex(osfs.New(repo.path, osfs.WithMmap())).
// That removes this function's body, commitGraphMemoryLimit and memoryFile.
// Reading the files instead costs up to +85% on a walk that bisects, as each
// lookup in the graph is then a dozen system calls or more.
func (repo *GoGitRepo) openCommitGraph() commitgraph.Index {
	info := filepath.Join(repo.path, "objects", "info")

	paths := []string{filepath.Join(info, "commit-graph")}
	if _, err := os.Stat(paths[0]); err != nil {
		f, err := os.Open(filepath.Join(info, "commit-graphs", "commit-graph-chain"))
		if err != nil {
			return nil
		}
		chain, err := commitgraph.OpenChainFile(f)
		_ = f.Close()
		if err != nil {
			return nil
		}
		paths = paths[:0]
		for _, h := range chain {
			paths = append(paths, filepath.Join(info, "commit-graphs", "graph-"+h+".graph"))
		}
	}

	var size int64
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			return nil
		}
		size += fi.Size()
	}
	open := func(p string) (commitgraph.ReaderAtCloser, error) {
		if size > commitGraphMemoryLimit {
			return os.Open(p)
		}
		b, err := os.ReadFile(p)
		return memoryFile{bytes.NewReader(b)}, err
	}

	// Each layer is opened on top of the previous one, and closing it closes
	// them all.
	var index commitgraph.Index
	fail := func() commitgraph.Index {
		if index != nil {
			_ = index.Close()
		}
		return nil
	}
	for _, p := range paths {
		r, err := open(p)
		if err != nil {
			return fail()
		}
		next, err := commitgraph.OpenFileIndexWithParent(r, index)
		if err != nil {
			_ = r.Close()
			return fail()
		}
		index = next
	}
	return index
}

// memoryFile is a file read in memory, for openCommitGraph.
type memoryFile struct {
	*bytes.Reader
}

func (memoryFile) Close() error { return nil }
