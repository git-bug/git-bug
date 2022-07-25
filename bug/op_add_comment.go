package bug

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/text"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

var _ Operation = &AddCommentOperation{}
var _ dag.OperationWithFiles = &AddCommentOperation{}

// AddCommentOperation will add a new comment in the bug
type AddCommentOperation struct {
	dag.OpBase
	Message string `json:"message"`
	// TODO: change for a map[string]util.hash to store the filename ?
	Files []repository.Hash `json:"files"`
}

func (op *AddCommentOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *AddCommentOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author())
	snapshot.addParticipant(op.Author())

	comment := Comment{
		id:       entity.CombineIds(snapshot.Id(), op.Id()),
		Message:  op.Message,
		Author:   op.Author(),
		Files:    op.Files,
		UnixTime: timestamp.Timestamp(op.UnixTime),
	}

	snapshot.Comments = append(snapshot.Comments, comment)

	item := &AddCommentTimelineItem{
		CommentTimelineItem: NewCommentTimelineItem(comment),
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *AddCommentOperation) GetFiles() []repository.Hash {
	return op.Files
}

func (op *AddCommentOperation) Validate() error {
	if err := op.OpBase.Validate(op, AddCommentOp); err != nil {
		return err
	}

	if !text.Safe(op.Message) {
		return fmt.Errorf("message is not fully printable")
	}

	return nil
}

func NewAddCommentOp(author identity.Interface, unixTime int64, message string, files []repository.Hash) *AddCommentOperation {
	return &AddCommentOperation{
		OpBase:  dag.NewOpBase(AddCommentOp, author, unixTime),
		Message: message,
		Files:   files,
	}
}

// AddCommentTimelineItem hold a comment in the timeline
type AddCommentTimelineItem struct {
	CommentTimelineItem
}

// IsAuthored is a sign post method for gqlgen
func (a *AddCommentTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func AddComment(b Interface, author identity.Interface, unixTime int64, message string) (*AddCommentOperation, error) {
	return AddCommentWithFiles(b, author, unixTime, message, nil)
}

func AddCommentWithFiles(b Interface, author identity.Interface, unixTime int64, message string, files []repository.Hash) (*AddCommentOperation, error) {
	addCommentOp := NewAddCommentOp(author, unixTime, message, files)
	if err := addCommentOp.Validate(); err != nil {
		return nil, err
	}
	b.Append(addCommentOp)
	return addCommentOp, nil
}
