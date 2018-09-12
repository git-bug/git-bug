package operations

import (
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/util/git"
)

// AddCommentOperation will add a new comment in the bug

var _ bug.Operation = AddCommentOperation{}

type AddCommentOperation struct {
	bug.OpBase
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

func NewAddCommentOp(author bug.Person, message string, files []git.Hash) AddCommentOperation {
	return AddCommentOperation{
		OpBase:  bug.NewOpBase(bug.AddCommentOp, author),
		Message: message,
		Files:   files,
	}
}

// Convenience function to apply the operation
func Comment(b bug.Interface, author bug.Person, message string) {
	CommentWithFiles(b, author, message, nil)
}

func CommentWithFiles(b bug.Interface, author bug.Person, message string, files []git.Hash) {
	addCommentOp := NewAddCommentOp(author, message, files)
	b.Append(addCommentOp)
}
