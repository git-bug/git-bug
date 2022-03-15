package bug

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/identity"
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
	dag.Operation

	// Type return the type of the operation
	Type() OperationType

	// Time return the time when the operation was added
	Time() time.Time
	// Apply the operation to a Snapshot to create the final state
	Apply(snapshot *Snapshot)

	// SetMetadata store arbitrary metadata about the operation
	SetMetadata(key string, value string)
	// GetMetadata retrieve arbitrary metadata about the operation
	GetMetadata(key string) (string, bool)
	// AllMetadata return all metadata for this operation
	AllMetadata() map[string]string

	setExtraMetadataImmutable(key string, value string)
}

func idOperation(op Operation, base *OpBase) entity.Id {
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

func operationUnmarshaller(author identity.Interface, raw json.RawMessage, resolver identity.Resolver) (dag.Operation, error) {
	var t struct {
		OperationType OperationType `json:"type"`
	}

	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}

	var op Operation

	switch t.OperationType {
	case AddCommentOp:
		op = &AddCommentOperation{}
	case CreateOp:
		op = &CreateOperation{}
	case EditCommentOp:
		op = &EditCommentOperation{}
	case LabelChangeOp:
		op = &LabelChangeOperation{}
	case NoOpOp:
		op = &NoOpOperation{}
	case SetMetadataOp:
		op = &SetMetadataOperation{}
	case SetStatusOp:
		op = &SetStatusOperation{}
	case SetTitleOp:
		op = &SetTitleOperation{}
	default:
		panic(fmt.Sprintf("unknown operation type %v", t.OperationType))
	}

	err := json.Unmarshal(raw, &op)
	if err != nil {
		return nil, err
	}

	switch op := op.(type) {
	case *AddCommentOperation:
		op.Author_ = author
	case *CreateOperation:
		op.Author_ = author
	case *EditCommentOperation:
		op.Author_ = author
	case *LabelChangeOperation:
		op.Author_ = author
	case *NoOpOperation:
		op.Author_ = author
	case *SetMetadataOperation:
		op.Author_ = author
	case *SetStatusOperation:
		op.Author_ = author
	case *SetTitleOperation:
		op.Author_ = author
	default:
		panic(fmt.Sprintf("unknown operation type %T", op))
	}

	return op, nil
}

// OpBase implement the common code for all operations
type OpBase struct {
	OperationType OperationType      `json:"type"`
	Author_       identity.Interface `json:"-"` // not serialized
	// TODO: part of the data model upgrade, this should eventually be a timestamp + lamport
	UnixTime int64             `json:"timestamp"`
	Metadata map[string]string `json:"metadata,omitempty"`

	// mandatory random bytes to ensure a better randomness of the data used to later generate the ID
	// len(Nonce) should be > 20 and < 64 bytes
	// It has no functional purpose and should be ignored.
	Nonce []byte `json:"nonce"`

	// Not serialized. Store the op's id in memory.
	id entity.Id
	// Not serialized. Store the extra metadata in memory,
	// compiled from SetMetadataOperation.
	extraMetadata map[string]string
}

// newOpBase is the constructor for an OpBase
func newOpBase(opType OperationType, author identity.Interface, unixTime int64) OpBase {
	return OpBase{
		OperationType: opType,
		Author_:       author,
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

func (base *OpBase) UnmarshalJSON(data []byte) error {
	// Compute the Id when loading the op from disk.
	base.id = entity.DeriveId(data)

	aux := struct {
		OperationType OperationType     `json:"type"`
		UnixTime      int64             `json:"timestamp"`
		Metadata      map[string]string `json:"metadata,omitempty"`
		Nonce         []byte            `json:"nonce"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	base.OperationType = aux.OperationType
	base.UnixTime = aux.UnixTime
	base.Metadata = aux.Metadata
	base.Nonce = aux.Nonce

	return nil
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
	if base.OperationType != opType {
		return fmt.Errorf("incorrect operation type (expected: %v, actual: %v)", opType, base.OperationType)
	}

	if op.Time().Unix() == 0 {
		return fmt.Errorf("time not set")
	}

	if base.Author_ == nil {
		return fmt.Errorf("author not set")
	}

	if err := op.Author().Validate(); err != nil {
		return errors.Wrap(err, "author")
	}

	if op, ok := op.(dag.OperationWithFiles); ok {
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

// SetMetadata store arbitrary metadata about the operation
func (base *OpBase) SetMetadata(key string, value string) {
	if base.Metadata == nil {
		base.Metadata = make(map[string]string)
	}

	base.Metadata[key] = value
	base.id = entity.UnsetId
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

func (base *OpBase) setExtraMetadataImmutable(key string, value string) {
	if base.extraMetadata == nil {
		base.extraMetadata = make(map[string]string)
	}
	if _, exist := base.extraMetadata[key]; !exist {
		base.extraMetadata[key] = value
	}
}

// Author return author identity
func (base *OpBase) Author() identity.Interface {
	return base.Author_
}
