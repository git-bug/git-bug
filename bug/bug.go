package bug

import (
	"errors"
	"fmt"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util"
	"strings"
)

const bugsRefPattern = "refs/bugs/"
const bugsRemoteRefPattern = "refs/remotes/%s/bugs/"
const opsEntryName = "ops"
const rootEntryName = "root"

const idLength = 40
const humanIdLength = 7

// Bug hold the data of a bug thread, organized in a way close to
// how it will be persisted inside Git. This is the data structure
// used for merge of two different version.
type Bug struct {
	// Id used as unique identifier
	id string

	lastCommit util.Hash
	rootPack   util.Hash

	// TODO: need a way to order bugs, probably a Lamport clock

	packs []OperationPack

	staging OperationPack
}

// Create a new Bug
func NewBug() *Bug {
	// No id yet
	return &Bug{}
}

// Find an existing Bug matching a prefix
func FindLocalBug(repo repository.Repo, prefix string) (*Bug, error) {
	ids, err := repo.ListIds(bugsRefPattern)

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

	return ReadLocalBug(repo, matching[0])
}

func ReadLocalBug(repo repository.Repo, id string) (*Bug, error) {
	ref := bugsRefPattern + id
	return readBug(repo, ref)
}

func ReadRemoteBug(repo repository.Repo, remote string, id string) (*Bug, error) {
	ref := fmt.Sprintf(bugsRemoteRefPattern, remote) + id
	return readBug(repo, ref)
}

// Read and parse a Bug from git
func readBug(repo repository.Repo, ref string) (*Bug, error) {
	hashes, err := repo.ListCommits(ref)

	if err != nil {
		return nil, err
	}

	refSplitted := strings.Split(ref, "/")
	id := refSplitted[len(refSplitted)-1]

	if len(id) != idLength {
		return nil, fmt.Errorf("Invalid ref length")
	}

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
			if entry.Name == opsEntryName {
				opsEntry = entry
				opsFound = true
				continue
			}
			if entry.Name == rootEntryName {
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

		if bug.rootPack == "" {
			bug.rootPack = rootEntry.Hash
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

type StreamedBug struct {
	Bug *Bug
	Err error
}

// Read and parse all local bugs
func ReadAllLocalBugs(repo repository.Repo) <-chan StreamedBug {
	return readAllBugs(repo, bugsRefPattern)
}

// Read and parse all remote bugs for a given remote
func ReadAllRemoteBugs(repo repository.Repo, remote string) <-chan StreamedBug {
	refPrefix := fmt.Sprintf(bugsRemoteRefPattern, remote)
	return readAllBugs(repo, refPrefix)
}

// Read and parse all available bug with a given ref prefix
func readAllBugs(repo repository.Repo, refPrefix string) <-chan StreamedBug {
	out := make(chan StreamedBug)

	go func() {
		defer close(out)

		refs, err := repo.ListRefs(refPrefix)
		if err != nil {
			out <- StreamedBug{Err: err}
			return
		}

		for _, ref := range refs {
			b, err := readBug(repo, ref)

			if err != nil {
				out <- StreamedBug{Err: err}
				return
			}

			out <- StreamedBug{Bug: b}
		}
	}()

	return out
}

// List all the available local bug ids
func ListLocalIds(repo repository.Repo) ([]string, error) {
	return repo.ListIds(bugsRefPattern)
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
		return fmt.Errorf("can't commit an empty bug")
	}

	// Write the Ops as a Git blob containing the serialized array
	hash, err := bug.staging.Write(repo)
	if err != nil {
		return err
	}

	if bug.rootPack == "" {
		bug.rootPack = hash
	}

	// Write a Git tree referencing this blob
	hash, err = repo.StoreTree([]repository.TreeEntry{
		// the last pack of ops
		{ObjectType: repository.Blob, Hash: hash, Name: opsEntryName},
		// always the first pack of ops (might be the same)
		{ObjectType: repository.Blob, Hash: bug.rootPack, Name: rootEntryName},
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

	// if it was the first commit, use the commit hash as bug id
	if bug.id == "" {
		bug.id = string(hash)
	}

	// Create or update the Git reference for this bug
	ref := fmt.Sprintf("%s%s", bugsRefPattern, bug.id)
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
	// Note: a faster merge should be possible without actually reading and parsing
	// all operations pack of our side.
	// Reading the other side is still necessary to validate remote data, at least
	// for new operations

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

	ancestorIndex := 0
	newPacks := make([]OperationPack, 0, len(bug.packs))

	// Find the root of the rebase
	for i, pack := range bug.packs {
		newPacks = append(newPacks, pack)

		if pack.commitHash == ancestor {
			ancestorIndex = i
			break
		}
	}

	if len(other.packs) == ancestorIndex+1 {
		// Nothing to rebase, return early
		return false, nil
	}

	// get other bug's extra packs
	for i := ancestorIndex + 1; i < len(other.packs); i++ {
		// clone is probably not necessary
		newPack := other.packs[i].Clone()

		newPacks = append(newPacks, newPack)
		bug.lastCommit = newPack.commitHash
	}

	// rebase our extra packs
	for i := ancestorIndex + 1; i < len(bug.packs); i++ {
		pack := bug.packs[i]

		// get the referenced git tree
		treeHash, err := repo.GetTreeHash(pack.commitHash)

		if err != nil {
			return false, err
		}

		// create a new commit with the correct ancestor
		hash, err := repo.StoreCommitWithParent(treeHash, bug.lastCommit)

		// replace the pack
		newPack := pack.Clone()
		newPack.commitHash = hash
		newPacks = append(newPacks, newPack)

		// update the bug
		bug.lastCommit = hash
	}

	// Update the git ref
	err = repo.UpdateRef(bugsRefPattern+bug.id, bug.lastCommit)
	if err != nil {
		return false, err
	}

	return true, nil
}

// Return the Bug identifier
func (bug *Bug) Id() string {
	if bug.id == "" {
		// simply panic as it would be a coding error
		// (using an id of a bug not stored yet)
		panic("no id yet")
	}
	return bug.id
}

// Return the Bug identifier truncated for human consumption
func (bug *Bug) HumanId() string {
	return formatHumanId(bug.Id())
}

func formatHumanId(id string) string {
	format := fmt.Sprintf("%%.%ds", humanIdLength)
	return fmt.Sprintf(format, id)
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
