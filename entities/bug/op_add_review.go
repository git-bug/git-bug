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

var _ Operation = &AddReviewOperation{}

// AddReviewOperation records a PR review: an approval, changes-requested, or
// plain commented verdict at a specific head commit.
// Valid only when the bug's Kind is PRKind.
type AddReviewOperation struct {
	dag.OpBase
	State      ReviewState `json:"state"`
	Body       string      `json:"body"`
	CommitHash string      `json:"commit_hash"`
}

func (op *AddReviewOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *AddReviewOperation) Apply(snapshot *Snapshot) {
	if snapshot.Kind != common.PRKind {
		return
	}
	snapshot.addActor(op.Author())
	snapshot.addParticipant(op.Author())

	opId := op.Id()
	review := Review{
		combinedId: entity.CombineIds(snapshot.Id(), opId),
		Author:     op.Author(),
		State:      op.State,
		Body:       op.Body,
		CommitHash: op.CommitHash,
		CreatedAt:  timestamp.Timestamp(op.UnixTime),
	}
	snapshot.Reviews = append(snapshot.Reviews, review)

	item := &AddReviewTimelineItem{
		combinedId: review.combinedId,
		Author:     op.Author(),
		UnixTime:   review.CreatedAt,
		State:      op.State,
		Body:       op.Body,
		CommitHash: op.CommitHash,
	}
	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *AddReviewOperation) Validate() error {
	if err := op.OpBase.Validate(op, AddReviewOp); err != nil {
		return err
	}
	if err := op.State.Validate(); err != nil {
		return fmt.Errorf("review state: %w", err)
	}
	if !text.Safe(op.Body) {
		return fmt.Errorf("review body is not fully printable")
	}
	if text.Empty(op.CommitHash) {
		return fmt.Errorf("commit_hash is empty")
	}
	if !text.SafeOneLine(op.CommitHash) {
		return fmt.Errorf("commit_hash has unsafe characters")
	}
	return nil
}

func NewAddReviewOp(author identity.Interface, unixTime int64, state ReviewState, body, commitHash string) *AddReviewOperation {
	return &AddReviewOperation{
		OpBase:     dag.NewOpBase(AddReviewOp, author, unixTime),
		State:      state,
		Body:       body,
		CommitHash: commitHash,
	}
}

type AddReviewTimelineItem struct {
	combinedId entity.CombinedId
	Author     identity.Interface
	UnixTime   timestamp.Timestamp
	State      ReviewState
	Body       string
	CommitHash string
}

func (a *AddReviewTimelineItem) CombinedId() entity.CombinedId { return a.combinedId }
func (a *AddReviewTimelineItem) IsAuthored()                   {}

// AddReview is a convenience function that appends a review to a PR.
func AddReview(b Interface, author identity.Interface, unixTime int64, state ReviewState, body, commitHash string, metadata map[string]string) (entity.CombinedId, *AddReviewOperation, error) {
	create, ok := b.FirstOp().(*CreateOperation)
	if !ok || create.Kind != common.PRKind {
		return entity.UnsetCombinedId, nil, fmt.Errorf("AddReview: bug is not a pull-request")
	}

	op := NewAddReviewOp(author, unixTime, state, body, commitHash)
	for k, v := range metadata {
		op.SetMetadata(k, v)
	}
	if err := op.Validate(); err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	b.Append(op)
	return entity.CombineIds(b.Id(), op.Id()), op, nil
}
