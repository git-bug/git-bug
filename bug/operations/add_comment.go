package operations

import "github.com/MichaelMure/git-bug/bug"

var _ bug.Operation = AddCommentOperation{}

type AddCommentOperation struct {
	bug.OpBase
	Message string     `json:"m"`
	Author  bug.Person `json:"a"`
}

func NewAddCommentOp(author bug.Person, message string) AddCommentOperation {
	return AddCommentOperation{
		OpBase:  bug.OpBase{OperationType: bug.ADD_COMMENT},
		Message: message,
		Author:  author,
	}
}

func (op AddCommentOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	comment := bug.Comment{
		Message: op.Message,
		Author:  op.Author,
	}

	snapshot.Comments = append(snapshot.Comments, comment)

	return snapshot
}
