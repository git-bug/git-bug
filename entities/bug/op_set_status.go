package bug

import (
	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/timestamp"
)

var _ Operation = &SetStatusOperation{}

// SetStatusOperation will change the status of a bug.
//
// MergeCommit is populated only when Status == MergedStatus (PR merges).
// It is ignored for any other status transition.
type SetStatusOperation struct {
	dag.OpBase
	Status      common.Status `json:"status"`
	MergeCommit string        `json:"merge_commit,omitempty"`
}

func (op *SetStatusOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *SetStatusOperation) Apply(snapshot *Snapshot) {
	snapshot.Status = op.Status
	snapshot.addActor(op.Author())
	if op.Status == common.MergedStatus {
		snapshot.MergeCommit = op.MergeCommit
	}

	id := op.Id()
	item := &SetStatusTimelineItem{
		// id:         id,
		combinedId:  entity.CombineIds(snapshot.Id(), id),
		Author:      op.Author(),
		UnixTime:    timestamp.Timestamp(op.UnixTime),
		Status:      op.Status,
		MergeCommit: op.MergeCommit,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *SetStatusOperation) Validate() error {
	if err := op.OpBase.Validate(op, SetStatusOp); err != nil {
		return err
	}

	if err := op.Status.Validate(); err != nil {
		return errors.Wrap(err, "status")
	}

	if op.Status == common.MergedStatus {
		if op.MergeCommit == "" {
			return errors.New("merge requires merge_commit")
		}
	} else if op.MergeCommit != "" {
		return errors.New("merge_commit only valid when status=merged")
	}

	return nil
}

func NewSetStatusOp(author identity.Interface, unixTime int64, status common.Status) *SetStatusOperation {
	return &SetStatusOperation{
		OpBase: dag.NewOpBase(SetStatusOp, author, unixTime),
		Status: status,
	}
}

// NewMergeOp builds a SetStatusOperation that marks a PR as merged and records
// the merge commit hash.
func NewMergeOp(author identity.Interface, unixTime int64, mergeCommit string) *SetStatusOperation {
	return &SetStatusOperation{
		OpBase:      dag.NewOpBase(SetStatusOp, author, unixTime),
		Status:      common.MergedStatus,
		MergeCommit: mergeCommit,
	}
}

type SetStatusTimelineItem struct {
	combinedId  entity.CombinedId
	Author      identity.Interface
	UnixTime    timestamp.Timestamp
	Status      common.Status
	MergeCommit string
}

func (s SetStatusTimelineItem) CombinedId() entity.CombinedId {
	return s.combinedId
}

// IsAuthored is a sign post-method for gqlgen, to mark compliance to an interface.
func (s *SetStatusTimelineItem) IsAuthored() {}

// Open is a convenience function to change a bugs state to Open
func Open(b Interface, author identity.Interface, unixTime int64, metadata map[string]string) (*SetStatusOperation, error) {
	op := NewSetStatusOp(author, unixTime, common.OpenStatus)
	for key, value := range metadata {
		op.SetMetadata(key, value)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}

// Close is a convenience function to change a bugs state to Close
func Close(b Interface, author identity.Interface, unixTime int64, metadata map[string]string) (*SetStatusOperation, error) {
	op := NewSetStatusOp(author, unixTime, common.ClosedStatus)
	for key, value := range metadata {
		op.SetMetadata(key, value)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}

// Merge is a convenience function to mark a PR as merged with the given commit hash.
func Merge(b Interface, author identity.Interface, unixTime int64, mergeCommit string, metadata map[string]string) (*SetStatusOperation, error) {
	op := NewMergeOp(author, unixTime, mergeCommit)
	for key, value := range metadata {
		op.SetMetadata(key, value)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}

// MarkReady is a convenience function to move a PR from draft to open (ready-for-review).
func MarkReady(b Interface, author identity.Interface, unixTime int64, metadata map[string]string) (*SetStatusOperation, error) {
	return Open(b, author, unixTime, metadata)
}
