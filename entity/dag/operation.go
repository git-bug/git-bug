package dag

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

// OperationType is an operation type identifier
type OperationType int

// Operation is a piece of data defining a change to reflect on the state of an Entity.
// What this Operation or Entity's state looks like is not of the resort of this package as it only deals with the
// data structure and storage.
type Operation interface {
	// Id return the Operation identifier
	//
	// Some care need to be taken to define a correct Id derivation and enough entropy in the data used to avoid
	// collisions. Notably:
	// - the Id of the first Operation will be used as the Id of the Entity. Collision need to be avoided across entities
	//   of the same type (example: no collision within the "bug" namespace).
	// - collisions can also happen within the set of Operations of an Entity. Simple Operation might not have enough
	//   entropy to yield unique Ids (example: two "close" operation within the same second, same author).
	//   If this is a concern, it is recommended to include a piece of random data in the operation's data, to guarantee
	//   a minimal amount of entropy and avoid collision.
	//
	//   Author's note: I tried to find a clever way around that inelegance (stuffing random useless data into the stored
	//   structure is not exactly elegant), but I failed to find a proper way. Essentially, anything that would reuse some
	//   other data (parent operation's Id, lamport clock) or the graph structure (depth) impose that the Id would only
	//   make sense in the context of the graph and yield some deep coupling between Entity and Operation. This in turn
	//   make the whole thing even less elegant.
	//
	// A common way to derive an Id will be to use the entity.DeriveId() function on the serialized operation data.
	Id() entity.Id
	// Type return the type of the operation
	Type() OperationType
	// Validate check if the Operation data is valid
	Validate() error
	// Author returns the author of this operation
	Author() identity.Interface
	// Time return the time when the operation was added
	Time() time.Time

	// SetMetadata store arbitrary metadata about the operation
	SetMetadata(key string, value string)
	// GetMetadata retrieve arbitrary metadata about the operation
	GetMetadata(key string) (string, bool)
	// AllMetadata return all metadata for this operation
	AllMetadata() map[string]string

	// setId allow to set the Id, used when unmarshalling only
	setId(id entity.Id)
	// setAuthor allow to set the author, used when unmarshalling only
	setAuthor(author identity.Interface)
	// setExtraMetadataImmutable add a metadata not carried by the operation itself on the operation
	setExtraMetadataImmutable(key string, value string)
}

type OperationWithApply[SnapT Snapshot] interface {
	Operation

	// Apply the operation to a Snapshot to create the final state
	Apply(snapshot SnapT)
}

// OperationWithFiles is an optional extension for an Operation that has files dependency, stored in git.
type OperationWithFiles interface {
	// GetFiles return the files needed by this operation
	// This implies that the Operation maintain and store internally the references to those files. This is how
	// this information is read later, when loading from storage.
	// For example, an operation that has a text value referencing some files would maintain a mapping (text ref -->
	// hash).
	GetFiles() []repository.Hash
}

// OperationDoesntChangeSnapshot is an interface signaling that the Operation implementing it doesn't change the
// snapshot, for example a metadata operation that act on other operations.
type OperationDoesntChangeSnapshot interface {
	DoesntChangeSnapshot()
}

// Snapshot is the minimal interface that a snapshot need to implement
type Snapshot interface {
	// AllOperations returns all the operations that have been applied to that snapshot, in order
	AllOperations() []Operation
	// AppendOperation add an operation in the list
	AppendOperation(op Operation)
}

// OpBase implement the common feature that every Operation should support.
type OpBase struct {
	// Not serialized. Store the op's id in memory.
	id entity.Id
	// Not serialized
	author identity.Interface

	OperationType OperationType `json:"type"`
	UnixTime      int64         `json:"timestamp"`

	// mandatory random bytes to ensure a better randomness of the data used to later generate the ID
	// len(Nonce) should be > 20 and < 64 bytes
	// It has no functional purpose and should be ignored.
	Nonce []byte `json:"nonce"`

	Metadata map[string]string `json:"metadata,omitempty"`
	// Not serialized. Store the extra metadata in memory,
	// compiled from SetMetadataOperation.
	extraMetadata map[string]string
}

func NewOpBase(opType OperationType, author identity.Interface, unixTime int64) OpBase {
	return OpBase{
		OperationType: opType,
		author:        author,
		UnixTime:      unixTime,
		Nonce:         makeNonce(20),
		id:            entity.UnsetId,
	}
}

func makeNonce(len int) []byte {
	result := make([]byte, len)
	_, err := rand.Read(result)
	if err != nil {
		panic(err)
	}
	return result
}

func IdOperation(op Operation, base *OpBase) entity.Id {
	if base.id == "" {
		// something went really wrong
		panic("op's id not set")
	}
	if base.id == entity.UnsetId {
		// This means we are trying to get the op's Id *before* it has been stored, for instance when
		// adding multiple ops in one go in an OperationPack.
		// As the Id is computed based on the actual bytes written on the disk, we are going to predict
		// those and then get the Id. This is safe as it will be the exact same code writing on disk later.

		data, err := json.Marshal(op)
		if err != nil {
			panic(err)
		}

		base.id = entity.DeriveId(data)
	}
	return base.id
}

func (base *OpBase) Type() OperationType {
	return base.OperationType
}

// Time return the time when the operation was added
func (base *OpBase) Time() time.Time {
	return time.Unix(base.UnixTime, 0)
}

// Validate check the OpBase for errors
func (base *OpBase) Validate(op Operation, opType OperationType) error {
	if base.OperationType == 0 {
		return fmt.Errorf("operation type unset")
	}
	if base.OperationType != opType {
		return fmt.Errorf("incorrect operation type (expected: %v, actual: %v)", opType, base.OperationType)
	}

	if op.Time().Unix() == 0 {
		return fmt.Errorf("time not set")
	}

	if base.author == nil {
		return fmt.Errorf("author not set")
	}

	if err := op.Author().Validate(); err != nil {
		return errors.Wrap(err, "author")
	}

	if op, ok := op.(OperationWithFiles); ok {
		for _, hash := range op.GetFiles() {
			if !hash.IsValid() {
				return fmt.Errorf("file with invalid hash %v", hash)
			}
		}
	}

	if len(base.Nonce) > 64 {
		return fmt.Errorf("nonce is too big")
	}
	if len(base.Nonce) < 20 {
		return fmt.Errorf("nonce is too small")
	}

	return nil
}

// IsAuthored is a sign post method for gqlgen
func (base *OpBase) IsAuthored() {}

// Author return author identity
func (base *OpBase) Author() identity.Interface {
	return base.author
}

// IdIsSet returns true if the id has been set already
func (base *OpBase) IdIsSet() bool {
	return base.id != "" && base.id != entity.UnsetId
}

// SetMetadata store arbitrary metadata about the operation
func (base *OpBase) SetMetadata(key string, value string) {
	if base.IdIsSet() {
		panic("set metadata on an operation with already an Id")
	}

	if base.Metadata == nil {
		base.Metadata = make(map[string]string)
	}
	base.Metadata[key] = value
}

// GetMetadata retrieve arbitrary metadata about the operation
func (base *OpBase) GetMetadata(key string) (string, bool) {
	val, ok := base.Metadata[key]

	if ok {
		return val, true
	}

	// extraMetadata can't replace the original operations value if any
	val, ok = base.extraMetadata[key]

	return val, ok
}

// AllMetadata return all metadata for this operation
func (base *OpBase) AllMetadata() map[string]string {
	result := make(map[string]string)

	for key, val := range base.extraMetadata {
		result[key] = val
	}

	// Original metadata take precedence
	for key, val := range base.Metadata {
		result[key] = val
	}

	return result
}

// setId allow to set the Id, used when unmarshalling only
func (base *OpBase) setId(id entity.Id) {
	if base.id != "" && base.id != entity.UnsetId {
		panic("trying to set id again")
	}
	base.id = id
}

// setAuthor allow to set the author, used when unmarshalling only
func (base *OpBase) setAuthor(author identity.Interface) {
	base.author = author
}

func (base *OpBase) setExtraMetadataImmutable(key string, value string) {
	if base.extraMetadata == nil {
		base.extraMetadata = make(map[string]string)
	}
	if _, exist := base.extraMetadata[key]; !exist {
		base.extraMetadata[key] = value
	}
}
