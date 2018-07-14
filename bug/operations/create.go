package operations

import (
	"github.com/MichaelMure/git-bug/bug"
	"reflect"
)

// CreateOperation define the initial creation of a bug

var _ bug.Operation = CreateOperation{}

type CreateOperation struct {
	bug.OpBase
	Title   string
	Message string
	Author  bug.Person
}

func NewCreateOp(author bug.Person, title, message string) CreateOperation {
	return CreateOperation{
		OpBase:  bug.OpBase{OperationType: bug.CREATE},
		Title:   title,
		Message: message,
		Author:  author,
	}
}

func (op CreateOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	empty := bug.Snapshot{}

	if !reflect.DeepEqual(snapshot, empty) {
		panic("Create operation should never be applied on a non-empty snapshot")
	}

	snapshot.Title = op.Title
	snapshot.Comments = []bug.Comment{
		{
			Message: op.Message,
			Author:  op.Author,
		},
	}
	return snapshot
}
