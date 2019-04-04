package bug

import (
	"encoding/json"
	"fmt"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/timestamp"

	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/text"
)

var _ Operation = &EditCommentOperation{}

// EditCommentOperation will change a comment in the bug
type EditCommentOperation struct {
	OpBase
	Target  git.Hash
	Message string
	Files   []git.Hash
}

func (op *EditCommentOperation) base() *OpBase {
	return &op.OpBase
}

func (op *EditCommentOperation) Hash() (git.Hash, error) {
	return hashOperation(op)
}

func (op *EditCommentOperation) Apply(snapshot *Snapshot) {
	// Todo: currently any message can be edited, even by a different author
	// crypto signature are needed.

	snapshot.addActor(op.Author)

	var target TimelineItem
	var commentIndex int

	for i, item := range snapshot.Timeline {
		h := item.Hash()

		if h == op.Target {
			target = snapshot.Timeline[i]
			break
		}

		// Track the index in the []Comment
		switch item.(type) {
		case *CreateTimelineItem, *CommentTimelineItem:
			commentIndex++
		}
	}

	if target == nil {
		// Target not found, edit is a no-op
		return
	}

	comment := Comment{
		id:       string(op.Target),
		Message:  op.Message,
		Files:    op.Files,
		UnixTime: timestamp.Timestamp(op.UnixTime),
	}

	switch target.(type) {
	case *CreateTimelineItem:
		item := target.(*CreateTimelineItem)
		item.Append(comment)

	case *AddCommentTimelineItem:
		item := target.(*AddCommentTimelineItem)
		item.Append(comment)
	}

	snapshot.Comments[commentIndex].Message = op.Message
	snapshot.Comments[commentIndex].Files = op.Files
}

func (op *EditCommentOperation) GetFiles() []git.Hash {
	return op.Files
}

func (op *EditCommentOperation) Validate() error {
	if err := opBaseValidate(op, EditCommentOp); err != nil {
		return err
	}

	if !op.Target.IsValid() {
		return fmt.Errorf("target hash is invalid")
	}

	if !text.Safe(op.Message) {
		return fmt.Errorf("message is not fully printable")
	}

	return nil
}

// Workaround to avoid the inner OpBase.MarshalJSON overriding the outer op
// MarshalJSON
func (op *EditCommentOperation) MarshalJSON() ([]byte, error) {
	base, err := json.Marshal(op.OpBase)
	if err != nil {
		return nil, err
	}

	// revert back to a flat map to be able to add our own fields
	var data map[string]interface{}
	if err := json.Unmarshal(base, &data); err != nil {
		return nil, err
	}

	data["target"] = op.Target
	data["message"] = op.Message
	data["files"] = op.Files

	return json.Marshal(data)
}

// Workaround to avoid the inner OpBase.MarshalJSON overriding the outer op
// MarshalJSON
func (op *EditCommentOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Target  git.Hash   `json:"target"`
		Message string     `json:"message"`
		Files   []git.Hash `json:"files"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Target = aux.Target
	op.Message = aux.Message
	op.Files = aux.Files

	return nil
}

// Sign post method for gqlgen
func (op *EditCommentOperation) IsAuthored() {}

func NewEditCommentOp(author identity.Interface, unixTime int64, target git.Hash, message string, files []git.Hash) *EditCommentOperation {
	return &EditCommentOperation{
		OpBase:  newOpBase(EditCommentOp, author, unixTime),
		Target:  target,
		Message: message,
		Files:   files,
	}
}

// Convenience function to apply the operation
func EditComment(b Interface, author identity.Interface, unixTime int64, target git.Hash, message string) (*EditCommentOperation, error) {
	return EditCommentWithFiles(b, author, unixTime, target, message, nil)
}

func EditCommentWithFiles(b Interface, author identity.Interface, unixTime int64, target git.Hash, message string, files []git.Hash) (*EditCommentOperation, error) {
	editCommentOp := NewEditCommentOp(author, unixTime, target, message, files)
	if err := editCommentOp.Validate(); err != nil {
		return nil, err
	}
	b.Append(editCommentOp)
	return editCommentOp, nil
}
