package bug

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util"
	"github.com/kevinburke/go.uuid"
	"strings"
)

const BugsRefPattern = "refs/bugs/"
const BugsRemoteRefPattern = "refs/remote/%s/bugs/"
const OpsEntryName = "ops"
const RootEntryName = "root"

// Bug hold the data of a bug thread, organized in a way close to
// how it will be persisted inside Git. This is the datastructure
// used for merge of two different version.
type Bug struct {
	// Id used as unique identifier
	id string

	lastCommit util.Hash
	root       util.Hash

	// TODO: need a way to order bugs, probably a Lamport clock

	packs []OperationPack

	staging OperationPack
}

// Create a new Bug
func NewBug() (*Bug, error) {
	// TODO: replace with commit hash of (first commit + some random)

	// Creating UUID Version 4
	unique, err := uuid.ID4()

	if err != nil {
		return nil, err
	}

	// Use it as source of uniqueness
	hash := sha256.New().Sum(unique.Bytes())

	// format in hex and truncate to 40 char
	id := fmt.Sprintf("%.40s", fmt.Sprintf("%x", hash))

	return &Bug{
		id: id,
	}, nil
}

// Find an existing Bug matching a prefix
func FindBug(repo repository.Repo, prefix string) (*Bug, error) {
	ids, err := repo.ListRefs(BugsRefPattern)

	if err != nil {
		return nil, err
	}

	// preallocate but empty
	matching := make([]string, 0, 5)

	for _, id := range ids {
		if strings.HasPrefix(id, prefix) {
			matching = append(matching, id)
		}
	}

	if len(matching) == 0 {
		return nil, errors.New("No matching bug found.")
	}

	if len(matching) > 1 {
		return nil, fmt.Errorf("Multiple matching bug found:\n%s", strings.Join(matching, "\n"))
	}

	return ReadBug(repo, BugsRefPattern+matching[0])
}

// Read and parse a Bug from git
func ReadBug(repo repository.Repo, ref string) (*Bug, error) {
	hashes, err := repo.ListCommits(ref)

	if err != nil {
		return nil, err
	}

	refSplitted := strings.Split(ref, "/")
	id := refSplitted[len(refSplitted)-1]

	bug := Bug{
		id: id,
	}

	// Load each OperationPack
	for _, hash := range hashes {
		entries, err := repo.ListEntries(hash)

		bug.lastCommit = hash

		if err != nil {
			return nil, err
		}

		var opsEntry repository.TreeEntry
		opsFound := false
		var rootEntry repository.TreeEntry
		rootFound := false

		for _, entry := range entries {
			if entry.Name == OpsEntryName {
				opsEntry = entry
				opsFound = true
				continue
			}
			if entry.Name == RootEntryName {
				rootEntry = entry
				rootFound = true
			}
		}

		if !opsFound {
			return nil, errors.New("Invalid tree, missing the ops entry")
		}

		if !rootFound {
			return nil, errors.New("Invalid tree, missing the root entry")
		}

		if bug.root == "" {
			bug.root = rootEntry.Hash
		}

		data, err := repo.ReadData(opsEntry.Hash)

		if err != nil {
			return nil, err
		}

		op, err := ParseOperationPack(data)

		if err != nil {
			return nil, err
		}

		// tag the pack with the commit hash
		op.commitHash = hash

		if err != nil {
			return nil, err
		}

		bug.packs = append(bug.packs, *op)
	}

	return &bug, nil
}

// IsValid check if the Bug data is valid
func (bug *Bug) IsValid() bool {
	// non-empty
	if len(bug.packs) == 0 && bug.staging.IsEmpty() {
		return false
	}

	// check if each pack is valid
	for _, pack := range bug.packs {
		if !pack.IsValid() {
			return false
		}
	}

	// check if staging is valid if needed
	if !bug.staging.IsEmpty() {
		if !bug.staging.IsValid() {
			return false
		}
	}

	// The very first Op should be a CreateOp
	firstOp := bug.firstOp()
	if firstOp == nil || firstOp.OpType() != CreateOp {
		return false
	}

	// Check that there is no more CreateOp op
	it := NewOperationIterator(bug)
	createCount := 0
	for it.Next() {
		if it.Value().OpType() == CreateOp {
			createCount++
		}
	}

	if createCount != 1 {
		return false
	}

	return true
}

func (bug *Bug) Append(op Operation) {
	bug.staging.Append(op)
}

// Write the staging area in Git and move the operations to the packs
func (bug *Bug) Commit(repo repository.Repo) error {
	if bug.staging.IsEmpty() {
		return nil
	}

	// Write the Ops as a Git blob containing the serialized array
	hash, err := bug.staging.Write(repo)
	if err != nil {
		return err
	}

	root := bug.root
	if root == "" {
		root = hash
		bug.root = hash
	}

	// Write a Git tree referencing this blob
	hash, err = repo.StoreTree([]repository.TreeEntry{
		{repository.Blob, hash, OpsEntryName},  // the last pack of ops
		{repository.Blob, root, RootEntryName}, // always the first pack of ops (might be the same)
	})
	if err != nil {
		return err
	}

	// Write a Git commit referencing the tree, with the previous commit as parent
	if bug.lastCommit != "" {
		hash, err = repo.StoreCommitWithParent(hash, bug.lastCommit)
	} else {
		hash, err = repo.StoreCommit(hash)
	}

	if err != nil {
		return err
	}

	bug.lastCommit = hash

	// Create or update the Git reference for this bug
	ref := fmt.Sprintf("%s%s", BugsRefPattern, bug.id)
	err = repo.UpdateRef(ref, hash)

	if err != nil {
		return err
	}

	bug.packs = append(bug.packs, bug.staging)
	bug.staging = OperationPack{}

	return nil
}

// Merge a different version of the same bug by rebasing operations of this bug
// that are not present in the other on top of the chain of operations of the
// other version.
func (bug *Bug) Merge(repo repository.Repo, other *Bug) (bool, error) {

	if bug.id != other.id {
		return false, errors.New("merging unrelated bugs is not supported")
	}

	if len(other.staging.Operations) > 0 {
		return false, errors.New("merging a bug with a non-empty staging is not supported")
	}

	if bug.lastCommit == "" || other.lastCommit == "" {
		return false, errors.New("can't merge a bug that has never been stored")
	}

	ancestor, err := repo.FindCommonAncestor(bug.lastCommit, other.lastCommit)

	if err != nil {
		return false, err
	}

	rebaseStarted := false
	updated := false

	for i, pack := range bug.packs {
		if pack.commitHash == ancestor {
			rebaseStarted = true

			// get other bug's extra pack
			for j := i + 1; j < len(other.packs); j++ {
				// clone is probably not necessary
				newPack := other.packs[j].Clone()

				bug.packs = append(bug.packs, newPack)
				bug.lastCommit = newPack.commitHash
				updated = true
			}

			continue
		}

		if !rebaseStarted {
			continue
		}

		updated = true

		// get the referenced git tree
		treeHash, err := repo.GetTreeHash(pack.commitHash)

		if err != nil {
			return false, err
		}

		// create a new commit with the correct ancestor
		hash, err := repo.StoreCommitWithParent(treeHash, bug.lastCommit)

		// replace the pack
		bug.packs[i] = pack.Clone()
		bug.packs[i].commitHash = hash

		// update the bug
		bug.lastCommit = hash
	}

	// Update the git ref
	if updated {
		err := repo.UpdateRef(BugsRefPattern+bug.id, bug.lastCommit)
		if err != nil {
			return false, err
		}
	}

	return updated, nil
}

// Return the Bug identifier
func (bug *Bug) Id() string {
	return bug.id
}

// Return the Bug identifier truncated for human consumption
func (bug *Bug) HumanId() string {
	return fmt.Sprintf("%.8s", bug.id)
}

// Lookup for the very first operation of the bug.
// For a valid Bug, this operation should be a CreateOp
func (bug *Bug) firstOp() Operation {
	for _, pack := range bug.packs {
		for _, op := range pack.Operations {
			return op
		}
	}

	if !bug.staging.IsEmpty() {
		return bug.staging.Operations[0]
	}

	return nil
}

// Compile a bug in a easily usable snapshot
func (bug *Bug) Compile() Snapshot {
	snap := Snapshot{
		id:     bug.id,
		Status: OpenStatus,
	}

	it := NewOperationIterator(bug)

	for it.Next() {
		op := it.Value()
		snap = op.Apply(snap)
		snap.Operations = append(snap.Operations, op)
	}

	return snap
}
