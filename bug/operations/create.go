package operations

import (
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/util"
)

// CreateOperation define the initial creation of a bug

var _ bug.Operation = CreateOperation{}

type CreateOperation struct {
	bug.OpBase
	Title   string
	Message string
	files   []util.Hash
}

func (op CreateOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	snapshot.Title = op.Title
	snapshot.Comments = []bug.Comment{
		{
			Message:  op.Message,
			Author:   op.Author,
			UnixTime: op.UnixTime(),
		},
	}
	snapshot.Author = op.Author
	snapshot.CreatedAt = op.Time()
	return snapshot
}

func (op CreateOperation) Files() []util.Hash {
	return op.files
}

func NewCreateOp(author bug.Person, title, message string, files []util.Hash) CreateOperation {
	return CreateOperation{
		OpBase:  bug.NewOpBase(bug.CreateOp, author),
		Title:   title,
		Message: message,
		files:   files,
	}
}

// Convenience function to apply the operation
func Create(author bug.Person, title, message string) (*bug.Bug, error) {
	return CreateWithFiles(author, title, message, nil)
}

func CreateWithFiles(author bug.Person, title, message string, files []util.Hash) (*bug.Bug, error) {
	newBug := bug.NewBug()
	createOp := NewCreateOp(author, title, message, files)
	newBug.Append(createOp)

	return newBug, nil
}
