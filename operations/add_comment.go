package operations

import (
	"fmt"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/text"
)

// AddCommentOperation will add a new comment in the bug

var _ bug.Operation = AddCommentOperation{}

type AddCommentOperation struct {
	*bug.OpBase
	Message string `json:"message"`
	// TODO: change for a map[string]util.hash to store the filename ?
	Files []git.Hash `json:"files"`
}

func (op AddCommentOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	comment := bug.Comment{
		Message:  op.Message,
		Author:   op.Author,
		Files:    op.Files,
		UnixTime: op.UnixTime,
	}

	snapshot.Comments = append(snapshot.Comments, comment)

	return snapshot
}

func (op AddCommentOperation) GetFiles() []git.Hash {
	return op.Files
}

func (op AddCommentOperation) Validate() error {
	if err := bug.OpBaseValidate(op, bug.AddCommentOp); err != nil {
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

func NewAddCommentOp(author bug.Person, unixTime int64, message string, files []git.Hash) AddCommentOperation {
	return AddCommentOperation{
		OpBase:  bug.NewOpBase(bug.AddCommentOp, author, unixTime),
		Message: message,
		Files:   files,
	}
}

// Convenience function to apply the operation
func Comment(b bug.Interface, author bug.Person, unixTime int64, message string) error {
	return CommentWithFiles(b, author, unixTime, message, nil)
}

func CommentWithFiles(b bug.Interface, author bug.Person, unixTime int64, message string, files []git.Hash) error {
	addCommentOp := NewAddCommentOp(author, unixTime, message, files)
	if err := addCommentOp.Validate(); err != nil {
		return err
	}
	b.Append(addCommentOp)
	return nil
}
