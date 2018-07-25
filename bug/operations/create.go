package operations

import (
	"github.com/MichaelMure/git-bug/bug"
)

// CreateOperation define the initial creation of a bug

var _ bug.Operation = CreateOperation{}

type CreateOperation struct {
	bug.OpBase
	Title   string
	Message string
}

func NewCreateOp(author bug.Person, title, message string) CreateOperation {
	return CreateOperation{
		OpBase:  bug.NewOpBase(bug.CreateOp, author),
		Title:   title,
		Message: message,
	}
}

func (op CreateOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	snapshot.Title = op.Title
	snapshot.Comments = []bug.Comment{
		{
			Message:  op.Message,
			Author:   op.Author,
			UnixTime: op.UnixTime,
		},
	}
	return snapshot
}

func Create(author bug.Person, title, message string) (*bug.Bug, error) {
	newBug := bug.NewBug()
	createOp := NewCreateOp(author, title, message)
	newBug.Append(createOp)

	return newBug, nil
}
