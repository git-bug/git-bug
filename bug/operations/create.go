package operations

import (
	"github.com/MichaelMure/git-bug/bug"
	"time"
)

// CreateOperation define the initial creation of a bug

var _ bug.Operation = CreateOperation{}

type CreateOperation struct {
	bug.OpBase
	Title   string
	Message string
	Author  bug.Person
	Time    int64
}

func NewCreateOp(author bug.Person, title, message string) CreateOperation {
	return CreateOperation{
		OpBase:  bug.OpBase{OperationType: bug.CreateOp},
		Title:   title,
		Message: message,
		Author:  author,
		Time:    time.Now().Unix(),
	}
}

func (op CreateOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	snapshot.Title = op.Title
	snapshot.Comments = []bug.Comment{
		{
			Message: op.Message,
			Author:  op.Author,
			Time:    op.Time,
		},
	}
	return snapshot
}
