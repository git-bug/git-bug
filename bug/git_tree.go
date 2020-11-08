package bug

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

type gitTree struct {
	opsEntry   repository.TreeEntry
	createTime lamport.Time
	editTime   lamport.Time
}

func readTree(repo repository.RepoData, hash repository.Hash) (*gitTree, error) {
	tree := &gitTree{}

	entries, err := repo.ReadTree(hash)
	if err != nil {
		return nil, errors.Wrap(err, "can't list git tree entries")
	}

	opsFound := false

	for _, entry := range entries {
		if entry.Name == opsEntryName {
			tree.opsEntry = entry
			opsFound = true
			continue
		}
		if strings.HasPrefix(entry.Name, createClockEntryPrefix) {
			n, err := fmt.Sscanf(entry.Name, createClockEntryPattern, &tree.createTime)
			if err != nil {
				return nil, errors.Wrap(err, "can't read create lamport time")
			}
			if n != 1 {
				return nil, fmt.Errorf("could not parse create time lamport value")
			}
		}
		if strings.HasPrefix(entry.Name, editClockEntryPrefix) {
			n, err := fmt.Sscanf(entry.Name, editClockEntryPattern, &tree.editTime)
			if err != nil {
				return nil, errors.Wrap(err, "can't read edit lamport time")
			}
			if n != 1 {
				return nil, fmt.Errorf("could not parse edit time lamport value")
			}
		}
	}

	if !opsFound {
		return nil, errors.New("invalid tree, missing the ops entry")
	}

	return tree, nil
}

func makeMediaTree(pack OperationPack) []repository.TreeEntry {
	var tree []repository.TreeEntry
	counter := 0
	added := make(map[repository.Hash]interface{})

	for _, ops := range pack.Operations {
		for _, file := range ops.GetFiles() {
			if _, has := added[file]; !has {
				tree = append(tree, repository.TreeEntry{
					ObjectType: repository.Blob,
					Hash:       file,
					// The name is not important here, we only need to
					// reference the blob.
					Name: fmt.Sprintf("file%d", counter),
				})
				counter++
				added[file] = struct{}{}
			}
		}
	}

	return tree
}
