package bug

import (
	"fmt"

	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/text"
)

var _ Operation = &EditCommentOperation{}

// EditCommentOperation will change a comment in the bug
type EditCommentOperation struct {
	OpBase
	Target  git.Hash   `json:"target"`
	Message string     `json:"message"`
	Files   []git.Hash `json:"files"`
}

func (op *EditCommentOperation) base() *OpBase {
	return &op.OpBase
}

func (op *EditCommentOperation) Hash() (git.Hash, error) {
	return hashOperation(op)
}

func (op *EditCommentOperation) Apply(snapshot *Snapshot) {
	// Todo: currently any message can be edited, even by a different author
	// crypto signature are needed.

	var target TimelineItem
	var commentIndex int

	for i, item := range snapshot.Timeline {
		h := item.Hash()

		if h == op.Target {
			target = snapshot.Timeline[i]
			break
		}

		// Track the index in the []Comment
		switch item.(type) {
		case *CreateTimelineItem, *CommentTimelineItem:
			commentIndex++
		}
	}

	if target == nil {
		// Target not found, edit is a no-op
		return
	}

	comment := Comment{
		Message:  op.Message,
		Files:    op.Files,
		UnixTime: Timestamp(op.UnixTime),
	}

	switch target.(type) {
	case *CreateTimelineItem:
		item := target.(*CreateTimelineItem)
		item.Append(comment)

	case *AddCommentTimelineItem:
		item := target.(*AddCommentTimelineItem)
		item.Append(comment)
	}

	snapshot.Comments[commentIndex].Message = op.Message
	snapshot.Comments[commentIndex].Files = op.Files
}

func (op *EditCommentOperation) GetFiles() []git.Hash {
	return op.Files
}

func (op *EditCommentOperation) Validate() error {
	if err := opBaseValidate(op, EditCommentOp); err != nil {
		return err
	}

	if !op.Target.IsValid() {
		return fmt.Errorf("target hash is invalid")
	}

	if text.Empty(op.Message) {
		return fmt.Errorf("message is empty")
	}

	if !text.Safe(op.Message) {
		return fmt.Errorf("message is not fully printable")
	}

	return nil
}

func NewEditCommentOp(author Person, unixTime int64, target git.Hash, message string, files []git.Hash) *EditCommentOperation {
	return &EditCommentOperation{
		OpBase:  newOpBase(EditCommentOp, author, unixTime),
		Target:  target,
		Message: message,
		Files:   files,
	}
}

// Convenience function to apply the operation
func EditComment(b Interface, author Person, unixTime int64, target git.Hash, message string) error {
	return EditCommentWithFiles(b, author, unixTime, target, message, nil)
}

func EditCommentWithFiles(b Interface, author Person, unixTime int64, target git.Hash, message string, files []git.Hash) error {
	editCommentOp := NewEditCommentOp(author, unixTime, target, message, files)
	if err := editCommentOp.Validate(); err != nil {
		return err
	}
	b.Append(editCommentOp)
	return nil
}
