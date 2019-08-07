package bug

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MichaelMure/git-bug/identity"

	"github.com/MichaelMure/git-bug/util/git"
	"github.com/pkg/errors"
)

// OperationType is an operation type identifier
type OperationType int

const (
	_ OperationType = iota
	CreateOp
	SetTitleOp
	AddCommentOp
	SetStatusOp
	LabelChangeOp
	EditCommentOp
	NoOpOp
	SetMetadataOp
)

const unsetIDMarker = "unset"

// Operation define the interface to fulfill for an edit operation of a Bug
type Operation interface {
	// base return the OpBase of the Operation, for package internal use
	base() *OpBase
	// ID return the identifier of the operation, to be used for back references
	ID() string
	// Time return the time when the operation was added
	Time() time.Time
	// GetUnixTime return the unix timestamp when the operation was added
	GetUnixTime() int64
	// GetFiles return the files needed by this operation
	GetFiles() []git.Hash
	// Apply the operation to a Snapshot to create the final state
	Apply(snapshot *Snapshot)
	// Validate check if the operation is valid (ex: a title is a single line)
	Validate() error
	// SetMetadata store arbitrary metadata about the operation
	SetMetadata(key string, value string)
	// GetMetadata retrieve arbitrary metadata about the operation
	GetMetadata(key string) (string, bool)
	// AllMetadata return all metadata for this operation
	AllMetadata() map[string]string
	// GetAuthor return the author identity
	GetAuthor() identity.Interface
}

func hashRaw(data []byte) string {
	hasher := sha256.New()
	// Write can't fail
	_, _ = hasher.Write(data)
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func idOperation(op Operation) string {
	base := op.base()

	if base.id == "" {
		// something went really wrong
		panic("op's id not set")
	}
	if base.id == "unset" {
		// This means we are trying to get the op's ID *before* it has been stored, for instance when
		// adding multiple ops in one go in an OperationPack.
		// As the ID is computed based on the actual bytes written on the disk, we are going to predict
		// those and then get the ID. This is safe as it will be the exact same code writing on disk later.

		data, err := json.Marshal(op)
		if err != nil {
			panic(err)
		}

		base.id = hashRaw(data)
	}
	return base.id
}

func IDIsValid(id string) bool {
	// IDs have the same format as a git hash
	if len(id) != 40 && len(id) != 64 {
		return false
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// OpBase implement the common code for all operations
type OpBase struct {
	OperationType OperationType
	Author        identity.Interface
	UnixTime      int64
	Metadata      map[string]string
	// Not serialized. Store the op's id in memory.
	id string
	// Not serialized. Store the extra metadata in memory,
	// compiled from SetMetadataOperation.
	extraMetadata map[string]string
}

// newOpBase is the constructor for an OpBase
func newOpBase(opType OperationType, author identity.Interface, unixTime int64) OpBase {
	return OpBase{
		OperationType: opType,
		Author:        author,
		UnixTime:      unixTime,
		id:            unsetIDMarker,
	}
}

func (op OpBase) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		OperationType OperationType      `json:"type"`
		Author        identity.Interface `json:"author"`
		UnixTime      int64              `json:"timestamp"`
		Metadata      map[string]string  `json:"metadata,omitempty"`
	}{
		OperationType: op.OperationType,
		Author:        op.Author,
		UnixTime:      op.UnixTime,
		Metadata:      op.Metadata,
	})
}

func (op *OpBase) UnmarshalJSON(data []byte) error {
	// Compute the ID when loading the op from disk.
	op.id = hashRaw(data)

	aux := struct {
		OperationType OperationType     `json:"type"`
		Author        json.RawMessage   `json:"author"`
		UnixTime      int64             `json:"timestamp"`
		Metadata      map[string]string `json:"metadata,omitempty"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// delegate the decoding of the identity
	author, err := identity.UnmarshalJSON(aux.Author)
	if err != nil {
		return err
	}

	op.OperationType = aux.OperationType
	op.Author = author
	op.UnixTime = aux.UnixTime
	op.Metadata = aux.Metadata

	return nil
}

// Time return the time when the operation was added
func (op *OpBase) Time() time.Time {
	return time.Unix(op.UnixTime, 0)
}

// GetUnixTime return the unix timestamp when the operation was added
func (op *OpBase) GetUnixTime() int64 {
	return op.UnixTime
}

// GetFiles return the files needed by this operation
func (op *OpBase) GetFiles() []git.Hash {
	return nil
}

// Validate check the OpBase for errors
func opBaseValidate(op Operation, opType OperationType) error {
	if op.base().OperationType != opType {
		return fmt.Errorf("incorrect operation type (expected: %v, actual: %v)", opType, op.base().OperationType)
	}

	if op.GetUnixTime() == 0 {
		return fmt.Errorf("time not set")
	}

	if op.base().Author == nil {
		return fmt.Errorf("author not set")
	}

	if err := op.base().Author.Validate(); err != nil {
		return errors.Wrap(err, "author")
	}

	for _, hash := range op.GetFiles() {
		if !hash.IsValid() {
			return fmt.Errorf("file with invalid hash %v", hash)
		}
	}

	return nil
}

// SetMetadata store arbitrary metadata about the operation
func (op *OpBase) SetMetadata(key string, value string) {
	if op.Metadata == nil {
		op.Metadata = make(map[string]string)
	}

	op.Metadata[key] = value
	op.id = unsetIDMarker
}

// GetMetadata retrieve arbitrary metadata about the operation
func (op *OpBase) GetMetadata(key string) (string, bool) {
	val, ok := op.Metadata[key]

	if ok {
		return val, true
	}

	// extraMetadata can't replace the original operations value if any
	val, ok = op.extraMetadata[key]

	return val, ok
}

// AllMetadata return all metadata for this operation
func (op *OpBase) AllMetadata() map[string]string {
	result := make(map[string]string)

	for key, val := range op.extraMetadata {
		result[key] = val
	}

	// Original metadata take precedence
	for key, val := range op.Metadata {
		result[key] = val
	}

	return result
}

// GetAuthor return author identity
func (op *OpBase) GetAuthor() identity.Interface {
	return op.Author
}
