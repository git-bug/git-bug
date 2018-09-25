package operations

import (
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/text"
)

// CreateOperation define the initial creation of a bug

var _ bug.Operation = CreateOperation{}

type CreateOperation struct {
	*bug.OpBase
	Title   string     `json:"title"`
	Message string     `json:"message"`
	Files   []git.Hash `json:"files"`
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
	snapshot.Author = op.Author
	snapshot.CreatedAt = op.Time()
	return snapshot
}

func (op CreateOperation) GetFiles() []git.Hash {
	return op.Files
}

func (op CreateOperation) Validate() error {
	if err := bug.OpBaseValidate(op, bug.CreateOp); err != nil {
		return err
	}

	if text.Empty(op.Title) {
		return fmt.Errorf("title is empty")
	}

	if strings.Contains(op.Title, "\n") {
		return fmt.Errorf("title should be a single line")
	}

	if !text.Safe(op.Title) {
		return fmt.Errorf("title is not fully printable")
	}

	if !text.Safe(op.Message) {
		return fmt.Errorf("message is not fully printable")
	}

	return nil
}

func NewCreateOp(author bug.Person, unixTime int64, title, message string, files []git.Hash) CreateOperation {
	return CreateOperation{
		OpBase:  bug.NewOpBase(bug.CreateOp, author, unixTime),
		Title:   title,
		Message: message,
		Files:   files,
	}
}

// Convenience function to apply the operation
func Create(author bug.Person, unixTime int64, title, message string) (*bug.Bug, error) {
	return CreateWithFiles(author, unixTime, title, message, nil)
}

func CreateWithFiles(author bug.Person, unixTime int64, title, message string, files []git.Hash) (*bug.Bug, error) {
	newBug := bug.NewBug()
	createOp := NewCreateOp(author, unixTime, title, message, files)

	if err := createOp.Validate(); err != nil {
		return nil, err
	}

	newBug.Append(createOp)

	return newBug, nil
}
