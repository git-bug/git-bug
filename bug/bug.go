// Package bug contains the bug data model and low-level related functions
package bug

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

const bugsRefPattern = "refs/bugs/"
const bugsRemoteRefPattern = "refs/remotes/%s/bugs/"

const opsEntryName = "ops"
const mediaEntryName = "media"

const createClockEntryPrefix = "create-clock-"
const createClockEntryPattern = "create-clock-%d"
const editClockEntryPrefix = "edit-clock-"
const editClockEntryPattern = "edit-clock-%d"

const creationClockName = "bug-create"
const editClockName = "bug-edit"

var ErrBugNotExist = errors.New("bug doesn't exist")

func NewErrMultipleMatchBug(matching []entity.Id) *entity.ErrMultipleMatch {
	return entity.NewErrMultipleMatch("bug", matching)
}

func NewErrMultipleMatchOp(matching []entity.Id) *entity.ErrMultipleMatch {
	return entity.NewErrMultipleMatch("operation", matching)
}

var _ Interface = &Bug{}
var _ entity.Interface = &Bug{}

// Bug hold the data of a bug thread, organized in a way close to
// how it will be persisted inside Git. This is the data structure
// used to merge two different version of the same Bug.
type Bug struct {
	// A Lamport clock is a logical clock that allow to order event
	// inside a distributed system.
	// It must be the first field in this struct due to https://github.com/golang/go/issues/599
	createTime lamport.Time
	editTime   lamport.Time

	lastCommit repository.Hash

	// all the committed operations
	packs []OperationPack

	// a temporary pack of operations used for convenience to pile up new operations
	// before a commit
	staging OperationPack
}

// NewBug create a new Bug
func NewBug() *Bug {
	// No logical clock yet
	return &Bug{}
}

// ReadLocal will read a local bug from its hash
func ReadLocal(repo repository.ClockedRepo, id entity.Id) (*Bug, error) {
	ref := bugsRefPattern + id.String()
	return read(repo, identity.NewSimpleResolver(repo), ref)
}

// ReadLocalWithResolver will read a local bug from its hash
func ReadLocalWithResolver(repo repository.ClockedRepo, identityResolver identity.Resolver, id entity.Id) (*Bug, error) {
	ref := bugsRefPattern + id.String()
	return read(repo, identityResolver, ref)
}

// ReadRemote will read a remote bug from its hash
func ReadRemote(repo repository.ClockedRepo, remote string, id entity.Id) (*Bug, error) {
	ref := fmt.Sprintf(bugsRemoteRefPattern, remote) + id.String()
	return read(repo, identity.NewSimpleResolver(repo), ref)
}

// ReadRemoteWithResolver will read a remote bug from its hash
func ReadRemoteWithResolver(repo repository.ClockedRepo, identityResolver identity.Resolver, remote string, id entity.Id) (*Bug, error) {
	ref := fmt.Sprintf(bugsRemoteRefPattern, remote) + id.String()
	return read(repo, identityResolver, ref)
}

// read will read and parse a Bug from git
func read(repo repository.ClockedRepo, identityResolver identity.Resolver, ref string) (*Bug, error) {
	id := entity.RefToId(ref)

	if err := id.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid ref ")
	}

	hashes, err := repo.ListCommits(ref)
	if err != nil {
		return nil, ErrBugNotExist
	}
	if len(hashes) == 0 {
		return nil, fmt.Errorf("empty bug")
	}

	bug := Bug{}

	// Load each OperationPack
	for _, hash := range hashes {
		tree, err := readTree(repo, hash)
		if err != nil {
			return nil, err
		}

		// Due to rebase, edit Lamport time are not necessarily ordered
		if tree.editTime > bug.editTime {
			bug.editTime = tree.editTime
		}

		// Update the clocks
		err = repo.Witness(creationClockName, bug.createTime)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update create lamport clock")
		}
		err = repo.Witness(editClockName, bug.editTime)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update edit lamport clock")
		}

		data, err := repo.ReadData(tree.opsEntry.Hash)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read git blob data")
		}

		opp := &OperationPack{}
		err = json.Unmarshal(data, &opp)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode OperationPack json")
		}

		// tag the pack with the commit hash
		opp.commitHash = hash
		bug.lastCommit = hash

		// if it's the first OperationPack read
		if len(bug.packs) == 0 {
			bug.createTime = tree.createTime
		}

		bug.packs = append(bug.packs, *opp)
	}

	// Bug Id is the Id of the first operation
	if len(bug.packs[0].Operations) == 0 {
		return nil, fmt.Errorf("first OperationPack is empty")
	}
	if id != bug.packs[0].Operations[0].Id() {
		return nil, fmt.Errorf("bug ID doesn't match the first operation ID")
	}

	// Make sure that the identities are properly loaded
	err = bug.EnsureIdentities(identityResolver)
	if err != nil {
		return nil, err
	}

	return &bug, nil
}

// RemoveBug will remove a local bug from its entity.Id
func RemoveBug(repo repository.ClockedRepo, id entity.Id) error {
	var fullMatches []string

	refs, err := repo.ListRefs(bugsRefPattern + id.String())
	if err != nil {
		return err
	}
	if len(refs) > 1 {
		return NewErrMultipleMatchBug(entity.RefsToIds(refs))
	}
	if len(refs) == 1 {
		// we have the bug locally
		fullMatches = append(fullMatches, refs[0])
	}

	remotes, err := repo.GetRemotes()
	if err != nil {
		return err
	}

	for remote := range remotes {
		remotePrefix := fmt.Sprintf(bugsRemoteRefPattern+id.String(), remote)
		remoteRefs, err := repo.ListRefs(remotePrefix)
		if err != nil {
			return err
		}
		if len(remoteRefs) > 1 {
			return NewErrMultipleMatchBug(entity.RefsToIds(refs))
		}
		if len(remoteRefs) == 1 {
			// found the bug in a remote
			fullMatches = append(fullMatches, remoteRefs[0])
		}
	}

	if len(fullMatches) == 0 {
		return ErrBugNotExist
	}

	for _, ref := range fullMatches {
		err = repo.RemoveRef(ref)
		if err != nil {
			return err
		}
	}

	return nil
}

type StreamedBug struct {
	Bug *Bug
	Err error
}

// ReadAllLocal read and parse all local bugs
func ReadAllLocal(repo repository.ClockedRepo) <-chan StreamedBug {
	return readAll(repo, identity.NewSimpleResolver(repo), bugsRefPattern)
}

// ReadAllLocalWithResolver read and parse all local bugs
func ReadAllLocalWithResolver(repo repository.ClockedRepo, identityResolver identity.Resolver) <-chan StreamedBug {
	return readAll(repo, identityResolver, bugsRefPattern)
}

// ReadAllRemote read and parse all remote bugs for a given remote
func ReadAllRemote(repo repository.ClockedRepo, remote string) <-chan StreamedBug {
	refPrefix := fmt.Sprintf(bugsRemoteRefPattern, remote)
	return readAll(repo, identity.NewSimpleResolver(repo), refPrefix)
}

// ReadAllRemoteWithResolver read and parse all remote bugs for a given remote
func ReadAllRemoteWithResolver(repo repository.ClockedRepo, identityResolver identity.Resolver, remote string) <-chan StreamedBug {
	refPrefix := fmt.Sprintf(bugsRemoteRefPattern, remote)
	return readAll(repo, identityResolver, refPrefix)
}

// Read and parse all available bug with a given ref prefix
func readAll(repo repository.ClockedRepo, identityResolver identity.Resolver, refPrefix string) <-chan StreamedBug {
	out := make(chan StreamedBug)

	go func() {
		defer close(out)

		refs, err := repo.ListRefs(refPrefix)
		if err != nil {
			out <- StreamedBug{Err: err}
			return
		}

		for _, ref := range refs {
			b, err := read(repo, identityResolver, ref)

			if err != nil {
				out <- StreamedBug{Err: err}
				return
			}

			out <- StreamedBug{Bug: b}
		}
	}()

	return out
}

// ListLocalIds list all the available local bug ids
func ListLocalIds(repo repository.Repo) ([]entity.Id, error) {
	refs, err := repo.ListRefs(bugsRefPattern)
	if err != nil {
		return nil, err
	}

	return entity.RefsToIds(refs), nil
}

// Validate check if the Bug data is valid
func (bug *Bug) Validate() error {
	// non-empty
	if len(bug.packs) == 0 && bug.staging.IsEmpty() {
		return fmt.Errorf("bug has no operations")
	}

	// check if each pack and operations are valid
	for _, pack := range bug.packs {
		if err := pack.Validate(); err != nil {
			return err
		}
	}

	// check if staging is valid if needed
	if !bug.staging.IsEmpty() {
		if err := bug.staging.Validate(); err != nil {
			return errors.Wrap(err, "staging")
		}
	}

	// The very first Op should be a CreateOp
	firstOp := bug.FirstOp()
	if firstOp == nil || firstOp.base().OperationType != CreateOp {
		return fmt.Errorf("first operation should be a Create op")
	}

	// Check that there is no more CreateOp op
	// Check that there is no colliding operation's ID
	it := NewOperationIterator(bug)
	createCount := 0
	ids := make(map[entity.Id]struct{})
	for it.Next() {
		if it.Value().base().OperationType == CreateOp {
			createCount++
		}
		if _, ok := ids[it.Value().Id()]; ok {
			return fmt.Errorf("id collision: %s", it.Value().Id())
		}
		ids[it.Value().Id()] = struct{}{}
	}

	if createCount != 1 {
		return fmt.Errorf("only one Create op allowed")
	}

	return nil
}

// Append an operation into the staging area, to be committed later
func (bug *Bug) Append(op Operation) {
	if len(bug.packs) == 0 && len(bug.staging.Operations) == 0 {
		if op.base().OperationType != CreateOp {
			panic("first operation should be a Create")
		}
	}
	bug.staging.Append(op)
}

// Commit write the staging area in Git and move the operations to the packs
func (bug *Bug) Commit(repo repository.ClockedRepo) error {
	if !bug.NeedCommit() {
		return fmt.Errorf("can't commit a bug with no pending operation")
	}

	if err := bug.Validate(); err != nil {
		return errors.Wrap(err, "can't commit a bug with invalid data")
	}

	// update clocks
	var err error
	bug.editTime, err = repo.Increment(editClockName)
	if err != nil {
		return err
	}
	if bug.lastCommit == "" {
		bug.createTime, err = repo.Increment(creationClockName)
		if err != nil {
			return err
		}
	}

	// Write the Ops as a Git blob containing the serialized array
	hash, err := bug.staging.Write(repo)
	if err != nil {
		return err
	}

	// Make a Git tree referencing this blob
	tree := []repository.TreeEntry{
		// the last pack of ops
		{ObjectType: repository.Blob, Hash: hash, Name: opsEntryName},
	}

	// Store the logical clocks as well
	// --> edit clock for each OperationPack/commits
	// --> create clock only for the first OperationPack/commits
	//
	// To avoid having one blob for each clock value, clocks are serialized
	// directly into the entry name
	emptyBlobHash, err := repo.StoreData([]byte{})
	if err != nil {
		return err
	}
	tree = append(tree, repository.TreeEntry{
		ObjectType: repository.Blob,
		Hash:       emptyBlobHash,
		Name:       fmt.Sprintf(editClockEntryPattern, bug.editTime),
	})
	if bug.lastCommit == "" {
		tree = append(tree, repository.TreeEntry{
			ObjectType: repository.Blob,
			Hash:       emptyBlobHash,
			Name:       fmt.Sprintf(createClockEntryPattern, bug.createTime),
		})
	}

	// Reference, if any, all the files required by the ops
	// Git will check that they actually exist in the storage and will make sure
	// to push/pull them as needed.
	mediaTree := makeMediaTree(bug.staging)
	if len(mediaTree) > 0 {
		mediaTreeHash, err := repo.StoreTree(mediaTree)
		if err != nil {
			return err
		}
		tree = append(tree, repository.TreeEntry{
			ObjectType: repository.Tree,
			Hash:       mediaTreeHash,
			Name:       mediaEntryName,
		})
	}

	// Store the tree
	hash, err = repo.StoreTree(tree)
	if err != nil {
		return err
	}

	// Write a Git commit referencing the tree, with the previous commit as parent
	if bug.lastCommit != "" {
		hash, err = repo.StoreCommit(hash, bug.lastCommit)
	} else {
		hash, err = repo.StoreCommit(hash)
	}
	if err != nil {
		return err
	}

	bug.lastCommit = hash
	bug.staging.commitHash = hash
	bug.packs = append(bug.packs, bug.staging)
	bug.staging = OperationPack{}

	// Create or update the Git reference for this bug
	// When pushing later, the remote will ensure that this ref update
	// is fast-forward, that is no data has been overwritten
	ref := fmt.Sprintf("%s%s", bugsRefPattern, bug.Id().String())
	return repo.UpdateRef(ref, hash)
}

func (bug *Bug) CommitAsNeeded(repo repository.ClockedRepo) error {
	if !bug.NeedCommit() {
		return nil
	}
	return bug.Commit(repo)
}

func (bug *Bug) NeedCommit() bool {
	return !bug.staging.IsEmpty()
}

// Merge a different version of the same bug by rebasing operations of this bug
// that are not present in the other on top of the chain of operations of the
// other version.
func (bug *Bug) Merge(repo repository.Repo, other Interface) (bool, error) {
	var otherBug = bugFromInterface(other)

	// Note: a faster merge should be possible without actually reading and parsing
	// all operations pack of our side.
	// Reading the other side is still necessary to validate remote data, at least
	// for new operations

	if bug.Id() != otherBug.Id() {
		return false, errors.New("merging unrelated bugs is not supported")
	}

	if len(otherBug.staging.Operations) > 0 {
		return false, errors.New("merging a bug with a non-empty staging is not supported")
	}

	if bug.lastCommit == "" || otherBug.lastCommit == "" {
		return false, errors.New("can't merge a bug that has never been stored")
	}

	ancestor, err := repo.FindCommonAncestor(bug.lastCommit, otherBug.lastCommit)
	if err != nil {
		return false, errors.Wrap(err, "can't find common ancestor")
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

	if len(otherBug.packs) == ancestorIndex+1 {
		// Nothing to rebase, return early
		return false, nil
	}

	// get other bug's extra packs
	for i := ancestorIndex + 1; i < len(otherBug.packs); i++ {
		// clone is probably not necessary
		newPack := otherBug.packs[i].Clone()

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
		hash, err := repo.StoreCommit(treeHash, bug.lastCommit)

		if err != nil {
			return false, err
		}

		// replace the pack
		newPack := pack.Clone()
		newPack.commitHash = hash
		newPacks = append(newPacks, newPack)

		// update the bug
		bug.lastCommit = hash
	}

	bug.packs = newPacks

	// Update the git ref
	err = repo.UpdateRef(bugsRefPattern+bug.Id().String(), bug.lastCommit)
	if err != nil {
		return false, err
	}

	return true, nil
}

// Id return the Bug identifier
func (bug *Bug) Id() entity.Id {
	// id is the id of the first operation
	return bug.FirstOp().Id()
}

// CreateLamportTime return the Lamport time of creation
func (bug *Bug) CreateLamportTime() lamport.Time {
	return bug.createTime
}

// EditLamportTime return the Lamport time of the last edit
func (bug *Bug) EditLamportTime() lamport.Time {
	return bug.editTime
}

// Lookup for the very first operation of the bug.
// For a valid Bug, this operation should be a CreateOp
func (bug *Bug) FirstOp() Operation {
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

// Lookup for the very last operation of the bug.
// For a valid Bug, should never be nil
func (bug *Bug) LastOp() Operation {
	if !bug.staging.IsEmpty() {
		return bug.staging.Operations[len(bug.staging.Operations)-1]
	}

	if len(bug.packs) == 0 {
		return nil
	}

	lastPack := bug.packs[len(bug.packs)-1]

	if len(lastPack.Operations) == 0 {
		return nil
	}

	return lastPack.Operations[len(lastPack.Operations)-1]
}

// Compile a bug in a easily usable snapshot
func (bug *Bug) Compile() Snapshot {
	snap := Snapshot{
		id:     bug.Id(),
		Status: OpenStatus,
	}

	it := NewOperationIterator(bug)

	for it.Next() {
		op := it.Value()
		op.Apply(&snap)
		snap.Operations = append(snap.Operations, op)
	}

	return snap
}
