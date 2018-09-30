package bug

import (
	"fmt"

	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/text"
)

var _ Operation = &AddCommentOperation{}

// AddCommentOperation will add a new comment in the bug
type AddCommentOperation struct {
	*OpBase
	Message string `json:"message"`
	// TODO: change for a map[string]util.hash to store the filename ?
	Files []git.Hash `json:"files"`
}

func (op *AddCommentOperation) base() *OpBase {
	return op.OpBase
}

func (op *AddCommentOperation) Hash() (git.Hash, error) {
	return hashOperation(op)
}

func (op *AddCommentOperation) Apply(snapshot *Snapshot) {
	comment := Comment{
		Message:  op.Message,
		Author:   op.Author,
		Files:    op.Files,
		UnixTime: Timestamp(op.UnixTime),
	}

	snapshot.Comments = append(snapshot.Comments, comment)

	hash, err := op.Hash()
	if err != nil {
		// Should never error unless a programming error happened
		// (covered in OpBase.Validate())
		panic(err)
	}

	snapshot.Timeline = append(snapshot.Timeline, NewCommentTimelineItem(hash, comment))
}

func (op *AddCommentOperation) GetFiles() []git.Hash {
	return op.Files
}

func (op *AddCommentOperation) Validate() error {
	if err := opBaseValidate(op, AddCommentOp); err != nil {
		return err
	}

	if text.Empty(op.Message) {
		return fmt.Errorf("message is empty")
	}

	if !text.Safe(op.Message) {
		return fmt.Errorf("message is not fully printable")
	}

	return nil
}

func NewAddCommentOp(author Person, unixTime int64, message string, files []git.Hash) *AddCommentOperation {
	return &AddCommentOperation{
		OpBase:  newOpBase(AddCommentOp, author, unixTime),
		Message: message,
		Files:   files,
	}
}

// Convenience function to apply the operation
func AddComment(b Interface, author Person, unixTime int64, message string) error {
	return AddCommentWithFiles(b, author, unixTime, message, nil)
}

func AddCommentWithFiles(b Interface, author Person, unixTime int64, message string, files []git.Hash) error {
	addCommentOp := NewAddCommentOp(author, unixTime, message, files)
	if err := addCommentOp.Validate(); err != nil {
		return err
	}
	b.Append(addCommentOp)
	return nil
}
