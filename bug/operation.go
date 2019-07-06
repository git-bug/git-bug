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

// Operation define the interface to fulfill for an edit operation of a Bug
type Operation interface {
	// base return the OpBase of the Operation, for package internal use
	base() *OpBase
	// Hash return the hash of the operation, to be used for back references
	Hash() (git.Hash, error)
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

func hashRaw(data []byte) git.Hash {
	hasher := sha256.New()
	// Write can't fail
	_, _ = hasher.Write(data)
	return git.Hash(fmt.Sprintf("%x", hasher.Sum(nil)))
}

// hash compute the hash of the serialized operation
func hashOperation(op Operation) (git.Hash, error) {
	// TODO: this might not be the best idea: if a single bit change in the output of json.Marshal, this will break.
	// Idea: hash the segment of serialized data (= immutable) instead of the go object in memory

	base := op.base()

	if base.hash != "" {
		return base.hash, nil
	}

	data, err := json.Marshal(op)
	if err != nil {
		return "", err
	}

	base.hash = hashRaw(data)

	return base.hash, nil
}

// OpBase implement the common code for all operations
type OpBase struct {
	OperationType OperationType
	Author        identity.Interface
	UnixTime      int64
	Metadata      map[string]string
	// Not serialized. Store the op's hash in memory.
	hash git.Hash
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
	op.hash = ""
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
