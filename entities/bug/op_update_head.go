package bug

import (
	"fmt"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/text"
	"github.com/git-bug/git-bug/util/timestamp"
)

var _ Operation = &UpdateHeadOperation{}

// UpdateHeadOperation records that a PR's head branch advanced to a new commit.
// Typical triggers: a push to the source branch, a force-push, or a rebase.
// Valid only when the bug's Kind is PRKind.
type UpdateHeadOperation struct {
	dag.OpBase
	// NewCommit is the new tip hash of the head branch.
	NewCommit string `json:"new_commit"`
	// PreviousCommit is the prior tip hash; recorded so the timeline can show
	// the diff and detect force-pushes.
	PreviousCommit string `json:"previous_commit"`
}

func (op *UpdateHeadOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *UpdateHeadOperation) Apply(snapshot *Snapshot) {
	if snapshot.Kind != common.PRKind {
		// Defence in depth: Validate rejects this, but Apply is called from
		// Compile which doesn't re-validate — stay idempotent and skip.
		return
	}
	snapshot.HeadCommit = op.NewCommit
	snapshot.addActor(op.Author())

	item := &UpdateHeadTimelineItem{
		combinedId:     entity.CombineIds(snapshot.Id(), op.Id()),
		Author:         op.Author(),
		UnixTime:       timestamp.Timestamp(op.UnixTime),
		NewCommit:      op.NewCommit,
		PreviousCommit: op.PreviousCommit,
	}
	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *UpdateHeadOperation) Validate() error {
	if err := op.OpBase.Validate(op, UpdateHeadOp); err != nil {
		return err
	}
	if text.Empty(op.NewCommit) {
		return fmt.Errorf("new_commit is empty")
	}
	if !text.SafeOneLine(op.NewCommit) {
		return fmt.Errorf("new_commit has unsafe characters")
	}
	if !text.SafeOneLine(op.PreviousCommit) {
		return fmt.Errorf("previous_commit has unsafe characters")
	}
	return nil
}

func NewUpdateHeadOp(author identity.Interface, unixTime int64, newCommit, previousCommit string) *UpdateHeadOperation {
	return &UpdateHeadOperation{
		OpBase:         dag.NewOpBase(UpdateHeadOp, author, unixTime),
		NewCommit:      newCommit,
		PreviousCommit: previousCommit,
	}
}

type UpdateHeadTimelineItem struct {
	combinedId     entity.CombinedId
	Author         identity.Interface
	UnixTime       timestamp.Timestamp
	NewCommit      string
	PreviousCommit string
}

func (u *UpdateHeadTimelineItem) CombinedId() entity.CombinedId { return u.combinedId }
func (u *UpdateHeadTimelineItem) IsAuthored()                   {}

// UpdateHead is a convenience function that advances the head commit of a PR.
// The bug must have been created as a PR (Kind == PRKind); otherwise returns an error.
func UpdateHead(b Interface, author identity.Interface, unixTime int64, newCommit string, metadata map[string]string) (*UpdateHeadOperation, error) {
	create, ok := b.FirstOp().(*CreateOperation)
	if !ok || create.Kind != common.PRKind {
		return nil, fmt.Errorf("UpdateHead: bug is not a pull-request")
	}

	// Find the most recent HeadCommit: last UpdateHeadOperation, else the create's HeadCommit.
	previous := create.HeadCommit
	for _, op := range b.Operations() {
		if uh, ok := op.(*UpdateHeadOperation); ok {
			previous = uh.NewCommit
		}
	}

	op := NewUpdateHeadOp(author, unixTime, newCommit, previous)
	for k, v := range metadata {
		op.SetMetadata(k, v)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}
