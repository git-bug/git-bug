package bug

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/timestamp"

	"github.com/MichaelMure/git-bug/util/text"
)

var _ Operation = &EditCommentOperation{}
var _ dag.OperationWithFiles = &EditCommentOperation{}

// EditCommentOperation will change a comment in the bug
type EditCommentOperation struct {
	OpBase
	Target  entity.Id         `json:"target"`
	Message string            `json:"message"`
	Files   []repository.Hash `json:"files"`
}

func (op *EditCommentOperation) Id() entity.Id {
	return idOperation(op, &op.OpBase)
}

func (op *EditCommentOperation) Apply(snapshot *Snapshot) {
	// Todo: currently any message can be edited, even by a different author
	// crypto signature are needed.

	// Recreate the Comment Id to match on
	commentId := entity.CombineIds(snapshot.Id(), op.Target)

	var target TimelineItem
	for i, item := range snapshot.Timeline {
		if item.Id() == commentId {
			target = snapshot.Timeline[i]
			break
		}
	}

	if target == nil {
		// Target not found, edit is a no-op
		return
	}

	comment := Comment{
		id:       commentId,
		Message:  op.Message,
		Files:    op.Files,
		UnixTime: timestamp.Timestamp(op.UnixTime),
	}

	switch target := target.(type) {
	case *CreateTimelineItem:
		target.Append(comment)
	case *AddCommentTimelineItem:
		target.Append(comment)
	default:
		// somehow, the target matched on something that is not a comment
		// we make the op a no-op
		return
	}

	snapshot.addActor(op.Author_)

	// Updating the corresponding comment

	for i := range snapshot.Comments {
		if snapshot.Comments[i].Id() == commentId {
			snapshot.Comments[i].Message = op.Message
			snapshot.Comments[i].Files = op.Files
			break
		}
	}
}

func (op *EditCommentOperation) GetFiles() []repository.Hash {
	return op.Files
}

func (op *EditCommentOperation) Validate() error {
	if err := op.OpBase.Validate(op, EditCommentOp); err != nil {
		return err
	}

	if err := op.Target.Validate(); err != nil {
		return errors.Wrap(err, "target hash is invalid")
	}

	if !text.Safe(op.Message) {
		return fmt.Errorf("message is not fully printable")
	}

	return nil
}

// UnmarshalJSON is a two step JSON unmarshalling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *EditCommentOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Target  entity.Id         `json:"target"`
		Message string            `json:"message"`
		Files   []repository.Hash `json:"files"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Target = aux.Target
	op.Message = aux.Message
	op.Files = aux.Files

	return nil
}

// Sign post method for gqlgen
func (op *EditCommentOperation) IsAuthored() {}

func NewEditCommentOp(author identity.Interface, unixTime int64, target entity.Id, message string, files []repository.Hash) *EditCommentOperation {
	return &EditCommentOperation{
		OpBase:  newOpBase(EditCommentOp, author, unixTime),
		Target:  target,
		Message: message,
		Files:   files,
	}
}

// Convenience function to apply the operation
func EditComment(b Interface, author identity.Interface, unixTime int64, target entity.Id, message string) (*EditCommentOperation, error) {
	return EditCommentWithFiles(b, author, unixTime, target, message, nil)
}

func EditCommentWithFiles(b Interface, author identity.Interface, unixTime int64, target entity.Id, message string, files []repository.Hash) (*EditCommentOperation, error) {
	editCommentOp := NewEditCommentOp(author, unixTime, target, message, files)
	if err := editCommentOp.Validate(); err != nil {
		return nil, err
	}
	b.Append(editCommentOp)
	return editCommentOp, nil
}

// Convenience function to edit the body of a bug (the first comment)
func EditCreateComment(b Interface, author identity.Interface, unixTime int64, message string) (*EditCommentOperation, error) {
	createOp := b.FirstOp().(*CreateOperation)
	return EditComment(b, author, unixTime, createOp.Id(), message)
}

// Convenience function to edit the body of a bug (the first comment)
func EditCreateCommentWithFiles(b Interface, author identity.Interface, unixTime int64, message string, files []repository.Hash) (*EditCommentOperation, error) {
	createOp := b.FirstOp().(*CreateOperation)
	return EditCommentWithFiles(b, author, unixTime, createOp.Id(), message, files)
}
