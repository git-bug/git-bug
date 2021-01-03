package dag

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/openpgp"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
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

// operationPack is a wrapper structure to store multiple operations in a single git blob.
// Additionally, it holds and store the metadata for those operations.
type operationPack struct {
	// An identifier, taken from a hash of the serialized Operations.
	id entity.Id

	// The author of the Operations. Must be the same author for all the Operations.
	Author identity.Interface
	// The list of Operation stored in the operationPack
	Operations []Operation
	// Encode the entity's logical time of creation across all entities of the same type.
	// Only exist on the root operationPack
	CreateTime lamport.Time
	// Encode the entity's logical time of last edition across all entities of the same type.
	// Exist on all operationPack
	EditTime lamport.Time
	// // Encode the operationPack's logical time of creation withing this entity.
	// // Exist on all operationPack
	// PackTime lamport.Time
}

func (opp *operationPack) Id() entity.Id {
	if opp.id == "" || opp.id == entity.UnsetId {
		// This means we are trying to get the opp's Id *before* it has been stored.
		// As the Id is computed based on the actual bytes written on the disk, we are going to predict
		// those and then get the Id. This is safe as it will be the exact same code writing on disk later.

		data, err := json.Marshal(opp)
		if err != nil {
			panic(err)
		}
		opp.id = entity.DeriveId(data)
	}

	return opp.id
}

func (opp *operationPack) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Author     identity.Interface `json:"author"`
		Operations []Operation        `json:"ops"`
	}{
		Author:     opp.Author,
		Operations: opp.Operations,
	})
}

func (opp *operationPack) Validate() error {
	if opp.Author == nil {
		return fmt.Errorf("missing author")
	}
	for _, op := range opp.Operations {
		if op.Author() != opp.Author {
			return fmt.Errorf("operation has different author than the operationPack's")
		}
	}
	if opp.EditTime == 0 {
		return fmt.Errorf("lamport edit time is zero")
	}
	return nil
}

func (opp *operationPack) Write(def Definition, repo repository.RepoData, parentCommit ...repository.Hash) (repository.Hash, error) {
	if err := opp.Validate(); err != nil {
		return "", err
	}

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

	// Write the Ops as a Git blob containing the serialized array of operations
	data, err := json.Marshal(opp)
	if err != nil {
		return "", err
	}

	// compute the Id while we have the serialized data
	opp.id = entity.DeriveId(data)

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
		// {ObjectType: repository.Blob, Hash: emptyBlobHash,
		// 	Name: fmt.Sprintf(packClockEntryPrefix+"%d", opp.PackTime)},
	}
	if opp.CreateTime > 0 {
		tree = append(tree, repository.TreeEntry{
			ObjectType: repository.Blob,
			Hash:       emptyBlobHash,
			Name:       fmt.Sprintf(createClockEntryPrefix+"%d", opp.CreateTime),
		})
	}

	// Store the tree
	treeHash, err := repo.StoreTree(tree)
	if err != nil {
		return "", err
	}

	// Write a Git commit referencing the tree, with the previous commit as parent
	// If we have keys, sign.
	var commitHash repository.Hash

	// Sign the commit if we have a key
	if opp.Author.SigningKey() != nil {
		commitHash, err = repo.StoreSignedCommit(treeHash, opp.Author.SigningKey().PGPEntity(), parentCommit...)
	} else {
		commitHash, err = repo.StoreCommit(treeHash, parentCommit...)
	}

	if err != nil {
		return "", err
	}

	return commitHash, nil
}

// readOperationPack read the operationPack encoded in git at the given Tree hash.
//
// Validity of the Lamport clocks is left for the caller to decide.
func readOperationPack(def Definition, repo repository.RepoData, commit repository.Commit) (*operationPack, error) {
	entries, err := repo.ReadTree(commit.TreeHash)
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
		return nil, entity.NewErrUnknowFormat(def.formatVersion)
	}
	if version != def.formatVersion {
		return nil, entity.NewErrInvalidFormat(version, def.formatVersion)
	}

	var id entity.Id
	var author identity.Interface
	var ops []Operation
	var createTime lamport.Time
	var editTime lamport.Time
	// var packTime lamport.Time

	for _, entry := range entries {
		switch {
		case entry.Name == opsEntryName:
			data, err := repo.ReadData(entry.Hash)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read git blob data")
			}
			ops, author, err = unmarshallPack(def, data)
			if err != nil {
				return nil, err
			}
			id = entity.DeriveId(data)

		case strings.HasPrefix(entry.Name, createClockEntryPrefix):
			v, err := strconv.ParseUint(strings.TrimPrefix(entry.Name, createClockEntryPrefix), 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "can't read creation lamport time")
			}
			createTime = lamport.Time(v)

		case strings.HasPrefix(entry.Name, editClockEntryPrefix):
			v, err := strconv.ParseUint(strings.TrimPrefix(entry.Name, editClockEntryPrefix), 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "can't read edit lamport time")
			}
			editTime = lamport.Time(v)

			// case strings.HasPrefix(entry.Name, packClockEntryPrefix):
			// 	found &= 1 << 3
			//
			// 	v, err := strconv.ParseUint(strings.TrimPrefix(entry.Name, packClockEntryPrefix), 10, 64)
			// 	if err != nil {
			// 		return nil, errors.Wrap(err, "can't read pack lamport time")
			// 	}
			// 	packTime = lamport.Time(v)
		}
	}

	// Verify signature if we expect one
	keys := author.ValidKeysAtTime(fmt.Sprintf(editClockPattern, def.namespace), editTime)
	if len(keys) > 0 {
		keyring := identity.PGPKeyring(keys)
		_, err = openpgp.CheckDetachedSignature(keyring, commit.SignedData, commit.Signature)
		if err != nil {
			return nil, fmt.Errorf("signature failure: %v", err)
		}
	}

	return &operationPack{
		id:         id,
		Author:     author,
		Operations: ops,
		CreateTime: createTime,
		EditTime:   editTime,
		// PackTime:   packTime,
	}, nil
}

// unmarshallPack delegate the unmarshalling of the Operation's JSON to the decoding
// function provided by the concrete entity. This gives access to the concrete type of each
// Operation.
func unmarshallPack(def Definition, data []byte) ([]Operation, identity.Interface, error) {
	aux := struct {
		Author     identity.IdentityStub `json:"author"`
		Operations []json.RawMessage     `json:"ops"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return nil, nil, err
	}

	if aux.Author.Id() == "" || aux.Author.Id() == entity.UnsetId {
		return nil, nil, fmt.Errorf("missing author")
	}

	author, err := def.identityResolver.ResolveIdentity(aux.Author.Id())
	if err != nil {
		return nil, nil, err
	}

	ops := make([]Operation, 0, len(aux.Operations))

	for _, raw := range aux.Operations {
		// delegate to specialized unmarshal function
		op, err := def.operationUnmarshaler(author, raw)
		if err != nil {
			return nil, nil, err
		}
		ops = append(ops, op)
	}

	return ops, author, nil
}
