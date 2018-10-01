package bug

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

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
}

func hashRaw(data []byte) git.Hash {
	hasher := sha256.New()
	// Write can't fail
	_, _ = hasher.Write(data)
	return git.Hash(fmt.Sprintf("%x", hasher.Sum(nil)))
}

// hash compute the hash of the serialized operation
func hashOperation(op Operation) (git.Hash, error) {
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
	OperationType OperationType `json:"type"`
	Author        Person        `json:"author"`
	UnixTime      int64         `json:"timestamp"`
	hash          git.Hash
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// newOpBase is the constructor for an OpBase
func newOpBase(opType OperationType, author Person, unixTime int64) OpBase {
	return OpBase{
		OperationType: opType,
		Author:        author,
		UnixTime:      unixTime,
	}
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

	if _, err := op.Hash(); err != nil {
		return errors.Wrap(err, "op is not serializable")
	}

	if op.GetUnixTime() == 0 {
		return fmt.Errorf("time not set")
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
	return val, ok
}
