package entity

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

// TODO: extra data tree
const extraEntryName = "extra"

const opsEntryName = "ops"
const versionEntryPrefix = "version-"
const createClockEntryPrefix = "create-clock-"
const editClockEntryPrefix = "edit-clock-"
const packClockEntryPrefix = "pack-clock-"

type operationPack struct {
	Operations []Operation
	// Encode the entity's logical time of creation across all entities of the same type.
	// Only exist on the root operationPack
	CreateTime lamport.Time
	// Encode the entity's logical time of last edition across all entities of the same type.
	// Exist on all operationPack
	EditTime lamport.Time
	// Encode the operationPack's logical time of creation withing this entity.
	// Exist on all operationPack
	PackTime lamport.Time
}

func (opp operationPack) write(def Definition, repo repository.RepoData) (repository.Hash, error) {
	// For different reason, we store the clocks and format version directly in the git tree.
	// Version has to be accessible before any attempt to decode to return early with a unique error.
	// Clocks could possibly be stored in the git blob but it's nice to separate data and metadata, and
	// we are storing something directly in the tree already so why not.
	//
	// To have a valid Tree, we point the "fake" entries to always the same value, the empty blob.
	emptyBlobHash, err := repo.StoreData([]byte{})
	if err != nil {
		return "", err
	}

	// Write the Ops as a Git blob containing the serialized array
	data, err := json.Marshal(struct {
		Operations []Operation `json:"ops"`
	}{
		Operations: opp.Operations,
	})
	if err != nil {
		return "", err
	}
	hash, err := repo.StoreData(data)
	if err != nil {
		return "", err
	}

	// Make a Git tree referencing this blob and encoding the other values:
	// - format version
	// - clocks
	tree := []repository.TreeEntry{
		{ObjectType: repository.Blob, Hash: emptyBlobHash,
			Name: fmt.Sprintf(versionEntryPrefix+"%d", def.formatVersion)},
		{ObjectType: repository.Blob, Hash: hash,
			Name: opsEntryName},
		{ObjectType: repository.Blob, Hash: emptyBlobHash,
			Name: fmt.Sprintf(editClockEntryPrefix+"%d", opp.EditTime)},
		{ObjectType: repository.Blob, Hash: emptyBlobHash,
			Name: fmt.Sprintf(packClockEntryPrefix+"%d", opp.PackTime)},
	}
	if opp.CreateTime > 0 {
		tree = append(tree, repository.TreeEntry{
			ObjectType: repository.Blob,
			Hash:       emptyBlobHash,
			Name:       fmt.Sprintf(createClockEntryPrefix+"%d", opp.CreateTime),
		})
	}

	// Store the tree
	return repo.StoreTree(tree)
}

// readOperationPack read the operationPack encoded in git at the given Tree hash.
//
// Validity of the Lamport clocks is left for the caller to decide.
func readOperationPack(def Definition, repo repository.RepoData, treeHash repository.Hash) (*operationPack, error) {
	entries, err := repo.ReadTree(treeHash)
	if err != nil {
		return nil, err
	}

	// check the format version first, fail early instead of trying to read something
	var version uint
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name, versionEntryPrefix) {
			v, err := strconv.ParseUint(strings.TrimPrefix(entry.Name, versionEntryPrefix), 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "can't read format version")
			}
			if v > 1<<12 {
				return nil, fmt.Errorf("format version too big")
			}
			version = uint(v)
			break
		}
	}
	if version == 0 {
		return nil, NewErrUnknowFormat(def.formatVersion)
	}
	if version != def.formatVersion {
		return nil, NewErrInvalidFormat(version, def.formatVersion)
	}

	var ops []Operation
	var createTime lamport.Time
	var editTime lamport.Time
	var packTime lamport.Time

	for _, entry := range entries {
		if entry.Name == opsEntryName {
			data, err := repo.ReadData(entry.Hash)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read git blob data")
			}

			ops, err = unmarshallOperations(def, data)
			if err != nil {
				return nil, err
			}
			continue
		}

		if strings.HasPrefix(entry.Name, createClockEntryPrefix) {
			v, err := strconv.ParseUint(strings.TrimPrefix(entry.Name, createClockEntryPrefix), 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "can't read creation lamport time")
			}
			createTime = lamport.Time(v)
			continue
		}

		if strings.HasPrefix(entry.Name, editClockEntryPrefix) {
			v, err := strconv.ParseUint(strings.TrimPrefix(entry.Name, editClockEntryPrefix), 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "can't read edit lamport time")
			}
			editTime = lamport.Time(v)
			continue
		}

		if strings.HasPrefix(entry.Name, packClockEntryPrefix) {
			v, err := strconv.ParseUint(strings.TrimPrefix(entry.Name, packClockEntryPrefix), 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "can't read pack lamport time")
			}
			packTime = lamport.Time(v)
			continue
		}
	}

	return &operationPack{
		Operations: ops,
		CreateTime: createTime,
		EditTime:   editTime,
		PackTime:   packTime,
	}, nil
}

// unmarshallOperations delegate the unmarshalling of the Operation's JSON to the decoding
// function provided by the concrete entity. This gives access to the concrete type of each
// Operation.
func unmarshallOperations(def Definition, data []byte) ([]Operation, error) {
	aux := struct {
		Operations []json.RawMessage `json:"ops"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return nil, err
	}

	ops := make([]Operation, 0, len(aux.Operations))

	for _, raw := range aux.Operations {
		// delegate to specialized unmarshal function
		op, err := def.operationUnmarshaler(raw)
		if err != nil {
			return nil, err
		}

		ops = append(ops, op)
	}

	return ops, nil
}
