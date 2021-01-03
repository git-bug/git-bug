package repository

import (
	"io"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/errors"
)

// nonNativeMerge is an implementation of a branch merge, for the case where
// the underlying git implementation doesn't support it natively.
func nonNativeMerge(repo RepoData, ref string, otherRef string, treeHashFn func() Hash) error {
	commit, err := repo.ResolveRef(ref)
	if err != nil {
		return err
	}

	otherCommit, err := repo.ResolveRef(otherRef)
	if err != nil {
		return err
	}

	if commit == otherCommit {
		// nothing to merge
		return nil
	}

	// fast-forward is possible if otherRef include ref

	otherCommits, err := repo.ListCommits(otherRef)
	if err != nil {
		return err
	}

	fastForwardPossible := false
	for _, hash := range otherCommits {
		if hash == commit {
			fastForwardPossible = true
			break
		}
	}

	if fastForwardPossible {
		return repo.UpdateRef(ref, otherCommit)
	}

	// fast-forward is not possible, we need to create a merge commit

	// we need a Tree to make the commit, an empty Tree will do
	emptyTreeHash, err := repo.StoreTree(nil)
	if err != nil {
		return err
	}

	newHash, err := repo.StoreCommit(emptyTreeHash, commit, otherCommit)
	if err != nil {
		return err
	}

	return repo.UpdateRef(ref, newHash)
}

// nonNativeListCommits is an implementation for ListCommits, for the case where
// the underlying git implementation doesn't support if natively.
func nonNativeListCommits(repo RepoData, ref string) ([]Hash, error) {
	var result []Hash

	stack := make([]Hash, 0, 32)
	visited := make(map[Hash]struct{})

	hash, err := repo.ResolveRef(ref)
	if err != nil {
		return nil, err
	}

	stack = append(stack, hash)

	for len(stack) > 0 {
		// pop
		hash := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if _, ok := visited[hash]; ok {
			continue
		}

		// mark as visited
		visited[hash] = struct{}{}
		result = append(result, hash)

		commit, err := repo.ReadCommit(hash)
		if err != nil {
			return nil, err
		}

		for _, parent := range commit.Parents {
			stack = append(stack, parent)
		}
	}

	// reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}

// deArmorSignature convert an armored (text serialized) signature into raw binary
func deArmorSignature(armoredSig io.Reader) (io.Reader, error) {
	block, err := armor.Decode(armoredSig)
	if err != nil {
		return nil, err
	}
	if block.Type != openpgp.SignatureType {
		return nil, errors.InvalidArgumentError("expected '" + openpgp.SignatureType + "', got: " + block.Type)
	}
	return block.Body, nil
}
