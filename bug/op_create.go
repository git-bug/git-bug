package bug

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/text"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

var _ Operation = &CreateOperation{}

// CreateOperation define the initial creation of a bug
type CreateOperation struct {
	OpBase
	// mandatory random bytes to ensure a better randomness of the data of the first
	// operation of a bug, used to later generate the ID
	// len(Nonce) should be > 20 and < 64 bytes
	Nonce   []byte            `json:"nonce"`
	Title   string            `json:"title"`
	Message string            `json:"message"`
	Files   []repository.Hash `json:"files"`
}

// Sign-post method for gqlgen
func (op *CreateOperation) IsOperation() {}

func (op *CreateOperation) base() *OpBase {
	return &op.OpBase
}

func (op *CreateOperation) Id() entity.Id {
	return idOperation(op)
}

// OVERRIDE
func (op *CreateOperation) SetMetadata(key string, value string) {
	// sanity check: we make sure we are not in the following scenario:
	// - the bug is created with a first operation
	// - Id() is used
	// - metadata are added, which will change the Id
	// - Id() is used again

	if op.id != entity.UnsetId {
		panic("usage of Id() after changing the first operation")
	}

	op.OpBase.SetMetadata(key, value)
}

func (op *CreateOperation) Apply(snapshot *Snapshot) {
	// sanity check: will fail when adding a second Create
	if snapshot.id != "" && snapshot.id != entity.UnsetId && snapshot.id != op.Id() {
		panic("adding a second Create operation")
	}

	snapshot.id = op.Id()

	snapshot.addActor(op.Author)
	snapshot.addParticipant(op.Author)

	snapshot.Title = op.Title

	commentId := entity.CombineIds(snapshot.Id(), op.Id())
	comment := Comment{
		id:       commentId,
		Message:  op.Message,
		Author:   op.Author,
		UnixTime: timestamp.Timestamp(op.UnixTime),
	}

	snapshot.Comments = []Comment{comment}
	snapshot.Author = op.Author
	snapshot.CreateTime = op.Time()

	snapshot.Timeline = []TimelineItem{
		&CreateTimelineItem{
			CommentTimelineItem: NewCommentTimelineItem(commentId, comment),
		},
	}
}

func (op *CreateOperation) GetFiles() []repository.Hash {
	return op.Files
}

func (op *CreateOperation) Validate() error {
	if err := opBaseValidate(op, CreateOp); err != nil {
		return err
	}

	if len(op.Nonce) > 64 {
		return fmt.Errorf("create nonce is too big")
	}
	if len(op.Nonce) < 20 {
		return fmt.Errorf("create nonce is too small")
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

// UnmarshalJSON is a two step JSON unmarshalling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *CreateOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Nonce   []byte            `json:"nonce"`
		Title   string            `json:"title"`
		Message string            `json:"message"`
		Files   []repository.Hash `json:"files"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Nonce = aux.Nonce
	op.Title = aux.Title
	op.Message = aux.Message
	op.Files = aux.Files

	return nil
}

// Sign post method for gqlgen
func (op *CreateOperation) IsAuthored() {}

func makeNonce(len int) []byte {
	result := make([]byte, len)
	_, err := rand.Read(result)
	if err != nil {
		panic(err)
	}
	return result
}

func NewCreateOp(author identity.Interface, unixTime int64, title, message string, files []repository.Hash) *CreateOperation {
	return &CreateOperation{
		OpBase:  newOpBase(CreateOp, author, unixTime),
		Nonce:   makeNonce(20),
		Title:   title,
		Message: message,
		Files:   files,
	}
}

// CreateTimelineItem replace a Create operation in the Timeline and hold its edition history
type CreateTimelineItem struct {
	CommentTimelineItem
}

// Sign post method for gqlgen
func (c *CreateTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func Create(author identity.Interface, unixTime int64, title, message string) (*Bug, *CreateOperation, error) {
	return CreateWithFiles(author, unixTime, title, message, nil)
}

func CreateWithFiles(author identity.Interface, unixTime int64, title, message string, files []repository.Hash) (*Bug, *CreateOperation, error) {
	newBug := NewBug()
	createOp := NewCreateOp(author, unixTime, title, message, files)

	if err := createOp.Validate(); err != nil {
		return nil, createOp, err
	}

	newBug.Append(createOp)

	return newBug, createOp, nil
}
