package operations

import (
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/util"
)

// AddCommentOperation will add a new comment in the bug

var _ bug.Operation = AddCommentOperation{}

type AddCommentOperation struct {
	bug.OpBase
	Message string
	// TODO: change for a map[string]util.hash to store the filename ?
	files []util.Hash
}

func (op AddCommentOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	comment := bug.Comment{
		Message:  op.Message,
		Author:   op.Author,
		Files:    op.files,
		UnixTime: op.UnixTime,
	}

	snapshot.Comments = append(snapshot.Comments, comment)

	return snapshot
}

func (op AddCommentOperation) Files() []util.Hash {
	return op.files
}

func NewAddCommentOp(author bug.Person, message string, files []util.Hash) AddCommentOperation {
	return AddCommentOperation{
		OpBase:  bug.NewOpBase(bug.AddCommentOp, author),
		Message: message,
		files:   files,
	}
}

// Convenience function to apply the operation
func Comment(b *bug.Bug, author bug.Person, message string) {
	CommentWithFiles(b, author, message, nil)
}

func CommentWithFiles(b *bug.Bug, author bug.Person, message string, files []util.Hash) {
	addCommentOp := NewAddCommentOp(author, message, files)
	b.Append(addCommentOp)
}
