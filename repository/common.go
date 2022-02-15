package repository

import (
	"io"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/errors"
)

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
