package bug

import (
	"encoding/json"
	"fmt"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/text"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

var _ Operation = &AddCommentOperation{}

// AddCommentOperation will add a new comment in the bug
type AddCommentOperation struct {
	OpBase
	Message string `json:"message"`
	// TODO: change for a map[string]util.hash to store the filename ?
	Files []git.Hash `json:"files"`
}

// Sign-post method for gqlgen
func (op *AddCommentOperation) IsOperation() {}

func (op *AddCommentOperation) base() *OpBase {
	return &op.OpBase
}

func (op *AddCommentOperation) Id() entity.Id {
	return idOperation(op)
}

func (op *AddCommentOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author)
	snapshot.addParticipant(op.Author)

	comment := Comment{
		id:       op.Id(),
		Message:  op.Message,
		Author:   op.Author,
		Files:    op.Files,
		UnixTime: timestamp.Timestamp(op.UnixTime),
	}

	snapshot.Comments = append(snapshot.Comments, comment)

	item := &AddCommentTimelineItem{
		CommentTimelineItem: NewCommentTimelineItem(op.Id(), comment),
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *AddCommentOperation) GetFiles() []git.Hash {
	return op.Files
}

func (op *AddCommentOperation) Validate() error {
	if err := opBaseValidate(op, AddCommentOp); err != nil {
		return err
	}

	if !text.Safe(op.Message) {
		return fmt.Errorf("message is not fully printable")
	}

	return nil
}

// UnmarshalJSON is a two step JSON unmarshaling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *AddCommentOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Message string     `json:"message"`
		Files   []git.Hash `json:"files"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Message = aux.Message
	op.Files = aux.Files

	return nil
}

// Sign post method for gqlgen
func (op *AddCommentOperation) IsAuthored() {}

func NewAddCommentOp(author identity.Interface, unixTime int64, message string, files []git.Hash) *AddCommentOperation {
	return &AddCommentOperation{
		OpBase:  newOpBase(AddCommentOp, author, unixTime),
		Message: message,
		Files:   files,
	}
}

// CreateTimelineItem replace a AddComment operation in the Timeline and hold its edition history
type AddCommentTimelineItem struct {
	CommentTimelineItem
}

// Sign post method for gqlgen
func (a *AddCommentTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func AddComment(b Interface, author identity.Interface, unixTime int64, message string) (*AddCommentOperation, error) {
	return AddCommentWithFiles(b, author, unixTime, message, nil)
}

func AddCommentWithFiles(b Interface, author identity.Interface, unixTime int64, message string, files []git.Hash) (*AddCommentOperation, error) {
	addCommentOp := NewAddCommentOp(author, unixTime, message, files)
	if err := addCommentOp.Validate(); err != nil {
		return nil, err
	}
	b.Append(addCommentOp)
	return addCommentOp, nil
}
