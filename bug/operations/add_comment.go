package operations

import (
	"github.com/MichaelMure/git-bug/bug"
)

// AddCommentOperation will add a new comment in the bug

var _ bug.Operation = AddCommentOperation{}

type AddCommentOperation struct {
	bug.OpBase
	Message string
}

func (op AddCommentOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	comment := bug.Comment{
		Message:  op.Message,
		Author:   op.Author,
		UnixTime: op.UnixTime,
	}

	snapshot.Comments = append(snapshot.Comments, comment)

	return snapshot
}

func NewAddCommentOp(author bug.Person, message string) AddCommentOperation {
	return AddCommentOperation{
		OpBase:  bug.NewOpBase(bug.AddCommentOp, author),
		Message: message,
	}
}

// Convenience function to apply the operation
func Comment(b *bug.Bug, author bug.Person, message string) {
	addCommentOp := NewAddCommentOp(author, message)
	b.Append(addCommentOp)
}
