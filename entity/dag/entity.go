// Package dag contains the base common code to define an entity stored
// in a chain of git objects, supporting actions like Push, Pull and Merge.
package dag

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

const creationClockPattern = "%s-create"
const editClockPattern = "%s-edit"

type OperationUnmarshaler func(raw json.RawMessage, resolver entity.Resolvers) (Operation, error)

// Definition hold the details defining one specialization of an Entity.
type Definition struct {
	// the name of the entity (bug, pull-request, ...), for human consumption
	Typename string
	// the Namespace in git references (bugs, prs, ...)
	Namespace string
	// a function decoding a JSON message into an Operation
	OperationUnmarshaler OperationUnmarshaler
	// the expected format version number, that can be used for data migration/upgrade
	FormatVersion uint
}

// Entity is a data structure stored in a chain of git objects, supporting actions like Push, Pull and Merge.
type Entity struct {
	// A Lamport clock is a logical clock that allow to order event
	// inside a distributed system.
	// It must be the first field in this struct due to https://github.com/golang/go/issues/36606
	createTime lamport.Time
	editTime   lamport.Time

	Definition

	// mu protects createTime, editTime, ops, staging and lastCommit. Writers hold
	// it during the whole commit, as commits are rare and reads are frequent.
	mu sync.RWMutex

	// operations that are already stored in the repository
	ops []Operation
	// operations not yet stored in the repository
	staging []Operation

	lastCommit repository.Hash
}

// New create an empty Entity
func New(definition Definition) *Entity {
	return &Entity{
		Definition: definition,
	}
}

// Read will read and decode a stored local Entity from a repository
func Read[EntityT entity.Interface](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers, id entity.Id) (EntityT, error) {
	if err := id.Validate(); err != nil {
		return *new(EntityT), errors.Wrap(err, "invalid id")
	}

	commit, err := repo.ResolveRef(def.Namespace, id.String())
	if errors.Is(err, repository.ErrNotFound) {
		return *new(EntityT), entity.NewErrNotFound(def.Typename)
	}
	if err != nil {
		return *new(EntityT), err
	}

	return read[EntityT](def, wrapper, repo, resolvers, id, commit)
}

// readTracking will read and decode an Entity from the tracking refs of a remote
func readTracking[EntityT entity.Interface](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers, remote string, id entity.Id) (EntityT, error) {
	if err := id.Validate(); err != nil {
		return *new(EntityT), errors.Wrap(err, "invalid id")
	}

	commit, err := repo.ResolveTrackingRef(remote, def.Namespace, id.String())
	if errors.Is(err, repository.ErrNotFound) {
		return *new(EntityT), entity.NewErrNotFound(def.Typename)
	}
	if err != nil {
		return *new(EntityT), err
	}

	return read[EntityT](def, wrapper, repo, resolvers, id, commit)
}

// read fetch from git and decode an Entity from its last commit, and make sure
// that it is the Entity with the given id.
func read[EntityT entity.Interface](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers, id entity.Id, lastCommit repository.Hash) (EntityT, error) {
	ops, createTime, editTime, err := readSince(def, repo, resolvers, id, lastCommit, "", 0)
	if err != nil {
		return *new(EntityT), err
	}

	return wrapper(&Entity{
		Definition: def,
		ops:        ops,
		lastCommit: lastCommit,
		createTime: createTime,
		editTime:   editTime,
	}), nil
}

// readSince fetch from git and decode the operations of an Entity from its last commit,
// and make sure that it is the Entity with the given id.
// If since is not empty, only the commits after it are read, and sinceEditTime must be
// its edit time. Those commits must all descend from it, otherwise an error is returned.
func readSince(def Definition, repo repository.ClockedRepo, resolvers entity.Resolvers, id entity.Id, lastCommit repository.Hash, since repository.Hash, sinceEditTime lamport.Time) ([]Operation, lamport.Time, lamport.Time, error) {
	// Perform a breadth-first search to discover the commits of the DAG, going back in time
	// up to the chronological root, or to since. Along the way, count the children of each
	// commit.

	queue := make([]repository.Hash, 0, 32)
	visited := make(map[repository.Hash]struct{})
	commits := make(map[repository.Hash]repository.Commit)
	children := make(map[repository.Hash]int)

	queue = append(queue, lastCommit)
	visited[lastCommit] = struct{}{}
	if since != "" {
		visited[since] = struct{}{}
	}

	for len(queue) > 0 {
		// pop
		hash := queue[0]
		queue = queue[1:]

		commit, err := repo.ReadCommit(hash)
		if err != nil {
			return nil, 0, 0, err
		}

		commits[hash] = commit

		for _, parent := range commit.Parents {
			children[parent]++
			if _, ok := visited[parent]; !ok {
				queue = append(queue, parent)
				// mark as visited
				visited[parent] = struct{}{}
			}
		}
	}

	// Topological sort with Kahn's algorithm: starting from the last commit, a commit is
	// taken only once all its children have been, so children come first. The root is
	// necessarily last, as all the other commits descend from it.

	topoOrder := make([]repository.Commit, 0, len(commits))
	ready := []repository.Hash{lastCommit}

	for len(ready) > 0 {
		// pop
		hash := ready[0]
		ready = ready[1:]

		commit := commits[hash]
		topoOrder = append(topoOrder, commit)

		for _, parent := range commit.Parents {
			if _, ok := commits[parent]; !ok {
				// since, not walked
				continue
			}
			children[parent]--
			if children[parent] == 0 {
				ready = append(ready, parent)
			}
		}
	}

	// Now, we can reverse this topological order and read the commits in an order where
	// we are sure to have read all the chronological ancestors when we read a commit.

	// Next step is to:
	// 1) read the operationPacks
	// 2) make sure that clocks causality respect the DAG topology.

	oppMap := make(map[repository.Hash]*operationPack)
	opIds := make(map[entity.Id]struct{})
	var opsCount int

	// editTimeOf returns the edit time of an already verified commit
	editTimeOf := func(hash repository.Hash) lamport.Time {
		if hash == since {
			return sinceEditTime
		}
		pack, ok := oppMap[hash]
		if !ok {
			panic("topological ordering failed")
		}
		return pack.EditTime
	}

	// isAncestor tells if target is an ancestor of the already verified commit from.
	// As the clocks have been verified to follow the DAG, ancestors have a strictly
	// lower edit time: only the commits with an edit time higher than the target's
	// can lead to it.
	isAncestor := func(target, from repository.Hash) bool {
		targetEditTime := editTimeOf(target)
		stack := []repository.Hash{from}
		seen := map[repository.Hash]struct{}{from: {}}
		for len(stack) > 0 {
			hash := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, parent := range commits[hash].Parents {
				if parent == target {
					return true
				}
				if _, ok := seen[parent]; ok {
					continue
				}
				seen[parent] = struct{}{}
				if editTimeOf(parent) > targetEditTime {
					stack = append(stack, parent)
				}
			}
		}
		return false
	}

	for i := len(topoOrder) - 1; i >= 0; i-- {
		commit := topoOrder[i]
		isFirstCommit := since == "" && i == len(topoOrder)-1
		isMerge := len(commit.Parents) > 1

		// Verify DAG structure: single chronological root, so only the root
		// can have no parents. Said otherwise, the DAG need to have exactly
		// one leaf.
		// When reading from since, there should be no root at all: reaching one
		// means that some commits don't descend from since.
		if !isFirstCommit && len(commit.Parents) == 0 {
			return nil, 0, 0, fmt.Errorf("multiple leafs in the entity DAG")
		}

		opp, err := readOperationPack(def, repo, resolvers, commit)
		if err != nil {
			return nil, 0, 0, err
		}

		err = opp.Validate()
		if err != nil {
			return nil, 0, 0, err
		}

		if isMerge && len(opp.Operations) > 0 {
			return nil, 0, 0, fmt.Errorf("merge commit cannot have operations")
		}

		// an operation can only be stored once
		for _, op := range opp.Operations {
			if _, ok := opIds[op.Id()]; ok {
				return nil, 0, 0, fmt.Errorf("duplicate operation %s", op.Id())
			}
			opIds[op.Id()] = struct{}{}
		}

		// Check that the create lamport clock is set (not checked in Validate() as it's optional)
		if isFirstCommit && opp.CreateTime <= 0 {
			return nil, 0, 0, fmt.Errorf("creation lamport time not set")
		}

		// The id of an Entity is the id of its first operation: make sure that
		// the ref actually points to the Entity it is named after.
		if isFirstCommit && (len(opp.Operations) == 0 || opp.Operations[0].Id() != id) {
			return nil, 0, 0, fmt.Errorf("the %s doesn't match its id %s", def.Typename, id)
		}

		// make sure that the lamport clocks causality match the DAG topology
		for _, parentHash := range commit.Parents {
			parentEditTime := editTimeOf(parentHash)

			if parentEditTime >= opp.EditTime {
				return nil, 0, 0, fmt.Errorf("lamport clock ordering doesn't match the DAG")
			}

			// to avoid an attack where clocks are pushed toward the uint64 rollover, make sure
			// that the clocks don't jump too far in the future
			// we ignore merge commits here to allow merging after a loooong time without breaking anything,
			// as long as there is one valid chain of small hops, it's fine.
			if !isMerge && opp.EditTime-parentEditTime > 1_000_000 {
				return nil, 0, 0, fmt.Errorf("lamport clock jumping too far in the future, likely an attack")
			}
		}

		// A merge joins diverged branches: none of its parents can be an ancestor
		// of another one, as git-bug would fast-forward instead.
		if isMerge {
			for i, parent := range commit.Parents {
				for j, other := range commit.Parents {
					if i != j && (parent == other || isAncestor(other, parent)) {
						return nil, 0, 0, fmt.Errorf("merge commit with a parent that is an ancestor of another")
					}
				}
			}
		}

		oppMap[commit.Hash] = opp
		opsCount += len(opp.Operations)
	}

	// The clocks are fine, we witness them
	for _, opp := range oppMap {
		err := witnessClock(repo, fmt.Sprintf(creationClockPattern, def.Namespace), opp.CreateTime)
		if err != nil {
			return nil, 0, 0, err
		}
		err = witnessClock(repo, fmt.Sprintf(editClockPattern, def.Namespace), opp.EditTime)
		if err != nil {
			return nil, 0, 0, err
		}
	}

	// Now that we know that the topological order and clocks are fine, we order the operationPacks
	// based on the logical clocks, entirely ignoring the DAG topology

	oppSlice := make([]*operationPack, 0, len(oppMap))
	for _, pack := range oppMap {
		oppSlice = append(oppSlice, pack)
	}
	sort.Slice(oppSlice, func(i, j int) bool {
		// Primary ordering with the EditTime.
		if oppSlice[i].EditTime != oppSlice[j].EditTime {
			return oppSlice[i].EditTime < oppSlice[j].EditTime
		}
		// We have equal EditTime, which means we have concurrent edition over different machines, and we
		// can't tell which one came first. So, what now? We still need a total ordering and the most stable possible.
		// As a secondary ordering, we can order based on a hash of the serialized Operations in the
		// operationPack. It doesn't carry much meaning but it's unbiased and hard to abuse.
		// This is a lexicographic ordering on the stringified ID.
		return oppSlice[i].Id() < oppSlice[j].Id()
	})

	// Now that we ordered the operationPacks, we have the order of the Operations

	ops := make([]Operation, 0, opsCount)
	var createTime lamport.Time
	var editTime lamport.Time
	for _, pack := range oppSlice {
		for _, operation := range pack.Operations {
			ops = append(ops, operation)
		}
		if pack.CreateTime > createTime {
			createTime = pack.CreateTime
		}
		if pack.EditTime > editTime {
			editTime = pack.EditTime
		}
	}

	return ops, createTime, editTime, nil
}

// readClockNoCheck fetch from git the clocks of an Entity from its last commit: the creation time of its
// root, and the edit time of its last commit, which is the highest of the Entity.
// Note: readClockNoCheck does not verify the integrity of the Entity and could return incorrect or incomplete
// clocks if so. If data integrity check is a requirement, a flow similar to read without actually reading/decoding
// operation blobs can be implemented instead.
func readClockNoCheck(repo repository.ClockedRepo, lastCommit repository.Hash) (createTime, editTime lamport.Time, err error) {
	commit, err := repo.ReadCommit(lastCommit)
	if err != nil {
		return 0, 0, err
	}

	createTime, editTime, err = readOperationPackClock(repo, commit)
	if err != nil {
		return 0, 0, err
	}

	// if we have more than one commit, we need to find the root to have the create time
	if len(commit.Parents) > 0 {
		for len(commit.Parents) > 0 {
			// The path to the root is irrelevant.
			commit, err = repo.ReadCommit(commit.Parents[0])
			if err != nil {
				return 0, 0, err
			}
		}
		createTime, _, err = readOperationPackClock(repo, commit)
		if err != nil {
			return 0, 0, err
		}
	}

	if createTime <= 0 {
		return 0, 0, fmt.Errorf("creation lamport time not set")
	}
	if editTime <= 0 {
		return 0, 0, fmt.Errorf("edit lamport time not set")
	}

	return createTime, editTime, nil
}

// ReadAll read and parse all local Entity
func ReadAll[EntityT entity.Interface](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers) <-chan entity.StreamedEntity[EntityT] {
	out := make(chan entity.StreamedEntity[EntityT])

	go func() {
		defer close(out)

		refs, err := repo.ListRefs(def.Namespace)
		if err != nil {
			out <- entity.StreamedEntity[EntityT]{Err: err}
			return
		}

		total := int64(len(refs))
		current := int64(1)

		for key, commit := range refs {
			e, err := read[EntityT](def, wrapper, repo, resolvers, entity.Id(key), commit)

			if err != nil {
				out <- entity.StreamedEntity[EntityT]{Err: err}
				return
			}

			out <- entity.StreamedEntity[EntityT]{
				Entity:        e,
				CurrentEntity: current,
				TotalEntities: total,
			}
			current++
		}
	}()

	return out
}

// readAllClocksNoCheck goes over all entities matching Definition and return the highest creation and edit time
// they hold, for the corresponding clocks to be rebuilt. Zero if there is no entity.
func readAllClocksNoCheck(def Definition, repo repository.ClockedRepo) (createTime, editTime lamport.Time, err error) {
	refs, err := repo.ListRefs(def.Namespace)
	if err != nil {
		return 0, 0, err
	}

	for _, commit := range refs {
		create, edit, err := readClockNoCheck(repo, commit)
		if err != nil {
			return 0, 0, err
		}
		createTime = max(createTime, create)
		editTime = max(editTime, edit)
	}

	return createTime, editTime, nil
}

// Id return the Entity identifier
func (e *Entity) Id() entity.Id {
	e.mu.RLock()
	defer e.mu.RUnlock()
	// id is the id of the first operation
	return e.firstOpLocked().Id()
}

// Validate check if the Entity data is valid
func (e *Entity) Validate() error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// non-empty
	if len(e.ops) == 0 && len(e.staging) == 0 {
		return fmt.Errorf("entity has no operations")
	}

	return validateOperations(e.ops, e.staging)
}

// validateOperations checks that each operation is valid, and that there is
// no colliding operation's ID across all the given lists.
func validateOperations(lists ...[]Operation) error {
	ids := make(map[entity.Id]struct{})
	for _, list := range lists {
		for _, op := range list {
			if err := op.Validate(); err != nil {
				return err
			}
			if _, ok := ids[op.Id()]; ok {
				return fmt.Errorf("id collision: %s", op.Id())
			}
			ids[op.Id()] = struct{}{}
		}
	}
	return nil
}

// Operations return the ordered operations.
// The returned slice must not be modified.
func (e *Entity) Operations() []Operation {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if len(e.staging) == 0 {
		// Committed operations are never modified in place, only appended, so they can be
		// shared. The capacity is capped so that appending to the result can't write into e.ops.
		return e.ops[:len(e.ops):len(e.ops)]
	}
	res := make([]Operation, 0, len(e.ops)+len(e.staging))
	res = append(res, e.ops...)
	return append(res, e.staging...)
}

// FirstOp lookup for the very first operation of the Entity
func (e *Entity) FirstOp() Operation {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.firstOpLocked()
}

func (e *Entity) firstOpLocked() Operation {
	for _, op := range e.ops {
		return op
	}
	for _, op := range e.staging {
		return op
	}
	return nil
}

// LastOp lookup for the very last operation of the Entity
func (e *Entity) LastOp() Operation {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if len(e.staging) > 0 {
		return e.staging[len(e.staging)-1]
	}
	if len(e.ops) > 0 {
		return e.ops[len(e.ops)-1]
	}
	return nil
}

// Append add a new Operation to the Entity
func (e *Entity) Append(op Operation) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.staging = append(e.staging, op)
}

// NeedCommit indicate if the in-memory state changed and need to be commit in the repository
func (e *Entity) NeedCommit() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.staging) > 0
}

// CommitAsNeeded execute a Commit only if necessary. This function is useful to avoid getting an error if the Entity
// is already in sync with the repository.
func (e *Entity) CommitAsNeeded(repo repository.ClockedRepo) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.staging) == 0 {
		return nil
	}
	return e.commitStagingLocked(repo)
}

// Commit write the appended operations in the repository.
// The Git reference is only moved if it still points to the last commit this Entity
// knows about, otherwise repository.ErrRefChanged is returned and Repair can be used
// before trying again. On any error, the Entity is left unchanged, with its operations
// still pending.
func (e *Entity) Commit(repo repository.ClockedRepo) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.staging) == 0 {
		return fmt.Errorf("can't commit an entity with no pending operation")
	}
	return e.commitStagingLocked(repo)
}

func (e *Entity) commitStagingLocked(repo repository.ClockedRepo) error {
	err := e.commitOperationsLocked(repo, e.staging)
	if err != nil {
		return err
	}
	e.staging = nil
	return nil
}

// CommitOperations writes the given operations in the repository, on top of the
// committed state of the Entity, without going through the pending operations,
// which it leaves untouched.
// The Git reference is only moved if it still points to the last commit this Entity
// knows about, otherwise repository.ErrRefChanged is returned and Repair can be used
// before trying again. On any error, the Entity is left unchanged.
// It can't be used while the first operation of the Entity is still pending, as
// that operation must stay the first one.
func (e *Entity) CommitOperations(repo repository.ClockedRepo, ops []Operation) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.ops) == 0 && len(e.staging) > 0 {
		return fmt.Errorf("can't commit operations before the pending first operation of a %s", e.Definition.Typename)
	}
	return e.commitOperationsLocked(repo, ops)
}

func (e *Entity) commitOperationsLocked(repo repository.ClockedRepo, ops []Operation) error {
	if len(ops) == 0 {
		return fmt.Errorf("can't commit an empty set of operations")
	}

	// pending operations are ignored, they are not part of this commit
	err := validateOperations(e.ops, ops)
	if err != nil {
		return errors.Wrapf(err, "can't commit a %s with invalid data", e.Definition.Typename)
	}

	// work on copies, the entity is only updated once the reference is
	createTime, editTime, lastCommit := e.createTime, e.editTime, e.lastCommit
	staging := ops

	for len(staging) > 0 {
		var author identity.Interface
		var toCommit []Operation

		// Split into chunks with the same author
		for len(staging) > 0 {
			op := staging[0]
			if author != nil && op.Author().Id() != author.Id() {
				break
			}
			author = op.Author()
			toCommit = append(toCommit, op)
			staging = staging[1:]
		}

		editTime, err = incrementClock(e.Definition, repo, editClockPattern)
		if err != nil {
			return err
		}

		opp := &operationPack{
			Author:     author,
			Operations: toCommit,
			EditTime:   editTime,
		}

		if lastCommit == "" {
			createTime, err = incrementClock(e.Definition, repo, creationClockPattern)
			if err != nil {
				return err
			}
			opp.CreateTime = createTime
		}

		var parentCommit []repository.Hash
		if lastCommit != "" {
			parentCommit = []repository.Hash{lastCommit}
		}

		lastCommit, err = opp.Write(e.Definition, repo, parentCommit...)
		if err != nil {
			return err
		}
	}

	// Create or update the Git reference for this entity
	// When pushing later, the remote will ensure that this ref update
	// is fast-forward, that is no data has been overwritten.
	// The id is the one of the first operation, which might not be committed yet.
	var id entity.Id
	if len(e.ops) > 0 {
		id = e.ops[0].Id()
	} else {
		id = ops[0].Id()
	}

	err = repo.UpdateRef(e.Namespace, id.String(), e.lastCommit, lastCommit)
	if err != nil {
		return err
	}

	e.createTime, e.editTime, e.lastCommit = createTime, editTime, lastCommit
	e.ops = append(e.ops, ops...)

	return nil
}

// Repair reloads the committed state of the Entity from the repository, typically
// after a commit failed with repository.ErrRefChanged because the reference has been
// moved by someone else. Pending operations are kept, and can be committed again on
// top of the reloaded state.
// When the new commits all descend from the last commit the Entity knows about, only
// those are read. Otherwise, the whole Entity is read again.
// On any error, the Entity is left unchanged.
func (e *Entity) Repair(repo repository.ClockedRepo, resolvers entity.Resolvers) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.ops) == 0 && len(e.staging) == 0 {
		return fmt.Errorf("can't repair an entity with no operations")
	}
	id := e.firstOpLocked().Id()

	commit, err := repo.ResolveRef(e.Namespace, id.String())
	if errors.Is(err, repository.ErrNotFound) {
		return entity.NewErrNotFound(e.Typename)
	}
	if err != nil {
		return err
	}

	if commit == e.lastCommit {
		return nil
	}

	// Incremental path: if all the new commits descend from the last known commit,
	// the Lamport clocks guarantee that all their operationPacks are ordered after
	// the known ones, which is the order a full read would give. Only the new commits
	// are then read, and their operations appended.
	// On error, not a descendant or not, fall back to the full read, which reports
	// any real problem.
	// Note: both read the clocks, so that the next commit get higher ones.
	if e.lastCommit != "" {
		ops, createTime, editTime, err := readSince(e.Definition, repo, resolvers, id, commit, e.lastCommit, e.editTime)
		if err == nil {
			// readSince only sees the new operations, check them against the known ones
			known := make(map[entity.Id]struct{}, len(e.ops))
			for _, op := range e.ops {
				known[op.Id()] = struct{}{}
			}
			for _, op := range ops {
				if _, ok := known[op.Id()]; ok {
					err = fmt.Errorf("duplicate operation %s", op.Id())
					break
				}
			}
		}
		if err == nil {
			e.ops = append(e.ops, ops...)
			e.createTime = max(e.createTime, createTime)
			e.editTime, e.lastCommit = editTime, commit
			return nil
		}
	}

	ops, createTime, editTime, err := readSince(e.Definition, repo, resolvers, id, commit, "", 0)
	if err != nil {
		return err
	}

	e.ops = ops
	e.createTime, e.editTime, e.lastCommit = createTime, editTime, commit

	return nil
}

// LastCommit returns the hash of the commit holding the last committed operations,
// that is what the Entity's reference points to as far as the Entity knows.
// It is empty if the Entity has never been committed.
func (e *Entity) LastCommit() repository.Hash {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastCommit
}

// CreateLamportTime return the Lamport time of creation
func (e *Entity) CreateLamportTime() lamport.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.createTime
}

// EditLamportTime return the Lamport time of the last edition
func (e *Entity) EditLamportTime() lamport.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.editTime
}
