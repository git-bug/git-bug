package bug

import (
	"fmt"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/text"
	"github.com/git-bug/git-bug/util/timestamp"
)

var _ Operation = &CreateOperation{}
var _ dag.OperationWithFiles = &CreateOperation{}

// CreateOperation define the initial creation of a bug.
//
// When Kind is PRKind, BaseRef/HeadRef identify the branches involved and
// HeadCommit is the commit hash at creation. These fields are empty and
// unused for plain issues; existing serialized issues predating PR support
// decode with the zero values (IssueKind + empty refs).
//
// The Go field is named Kind to avoid shadowing dag.OpBase.Type(). On the
// wire it serializes as "kind" — the JSON key "type" is already taken by
// dag.OpBase for the operation-type enum.
type CreateOperation struct {
	dag.OpBase
	Title   string            `json:"title"`
	Message string            `json:"message"`
	Files   []repository.Hash `json:"files"`

	Kind       common.Kind `json:"kind,omitempty"`
	BaseRef    string      `json:"base_ref,omitempty"`
	HeadRef    string      `json:"head_ref,omitempty"`
	HeadCommit string      `json:"head_commit,omitempty"`
	// Draft indicates a PR created in draft state. Ignored for issues.
	Draft bool `json:"draft,omitempty"`
}

func (op *CreateOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *CreateOperation) Apply(snapshot *Snapshot) {
	// sanity check: will fail when adding a second Create
	if snapshot.id != "" && snapshot.id != entity.UnsetId && snapshot.id != op.Id() {
		return
	}

	// the Id of the Bug/Snapshot is the Id of the first Operation: CreateOperation
	opId := op.Id()
	snapshot.id = opId

	snapshot.addActor(op.Author())
	snapshot.addParticipant(op.Author())

	snapshot.Title = op.Title
	snapshot.Kind = op.Kind

	if op.Kind == common.PRKind {
		snapshot.BaseRef = op.BaseRef
		snapshot.HeadRef = op.HeadRef
		snapshot.HeadCommit = op.HeadCommit
		if op.Draft {
			snapshot.Status = common.DraftStatus
		}
	}

	comment := Comment{
		combinedId: entity.CombineIds(snapshot.id, opId),
		targetId:   opId,
		Message:    op.Message,
		Author:     op.Author(),
		unixTime:   timestamp.Timestamp(op.UnixTime),
	}

	snapshot.Comments = []Comment{comment}
	snapshot.Author = op.Author()
	snapshot.CreateTime = op.Time()

	snapshot.Timeline = []TimelineItem{
		&CreateTimelineItem{
			CommentTimelineItem: NewCommentTimelineItem(comment),
		},
	}
}

func (op *CreateOperation) GetFiles() []repository.Hash {
	return op.Files
}

func (op *CreateOperation) Validate() error {
	if err := op.OpBase.Validate(op, CreateOp); err != nil {
		return err
	}

	if text.Empty(op.Title) {
		return fmt.Errorf("title is empty")
	}
	if !text.SafeOneLine(op.Title) {
		return fmt.Errorf("title has unsafe characters")
	}

	if !text.Safe(op.Message) {
		return fmt.Errorf("message is not fully printable")
	}

	if err := op.Kind.Validate(); err != nil {
		return fmt.Errorf("type: %w", err)
	}

	switch op.Kind {
	case common.IssueKind:
		if op.BaseRef != "" || op.HeadRef != "" || op.HeadCommit != "" || op.Draft {
			return fmt.Errorf("issue must not carry PR fields")
		}
	case common.PRKind:
		if text.Empty(op.BaseRef) {
			return fmt.Errorf("pr base_ref is empty")
		}
		if text.Empty(op.HeadRef) {
			return fmt.Errorf("pr head_ref is empty")
		}
		if !text.SafeOneLine(op.BaseRef) {
			return fmt.Errorf("pr base_ref has unsafe characters")
		}
		if !text.SafeOneLine(op.HeadRef) {
			return fmt.Errorf("pr head_ref has unsafe characters")
		}
		if !text.SafeOneLine(op.HeadCommit) {
			return fmt.Errorf("pr head_commit has unsafe characters")
		}
	}

	return nil
}

func NewCreateOp(author identity.Interface, unixTime int64, title, message string, files []repository.Hash) *CreateOperation {
	return &CreateOperation{
		OpBase:  dag.NewOpBase(CreateOp, author, unixTime),
		Title:   title,
		Message: message,
		Files:   files,
		Kind:    common.IssueKind,
	}
}

// NewCreatePROp builds a CreateOperation for a pull-request.
func NewCreatePROp(author identity.Interface, unixTime int64, title, message, baseRef, headRef, headCommit string, draft bool, files []repository.Hash) *CreateOperation {
	return &CreateOperation{
		OpBase:     dag.NewOpBase(CreateOp, author, unixTime),
		Title:      title,
		Message:    message,
		Files:      files,
		Kind:       common.PRKind,
		BaseRef:    baseRef,
		HeadRef:    headRef,
		HeadCommit: headCommit,
		Draft:      draft,
	}
}

// CreateTimelineItem replace a Create operation in the Timeline and hold its edition history
type CreateTimelineItem struct {
	CommentTimelineItem
}

// IsAuthored is a sign post-method for gqlgen, to mark compliance to an interface.
func (c *CreateTimelineItem) IsAuthored() {}

// Create is a convenience function to create a bug
func Create(author identity.Interface, unixTime int64, title, message string, files []repository.Hash, metadata map[string]string) (*Bug, *CreateOperation, error) {
	b := NewBug()
	op := NewCreateOp(author, unixTime, title, message, files)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, op, err
	}
	b.Append(op)
	return b, op, nil
}

// CreatePR is a convenience function to create a pull-request.
func CreatePR(author identity.Interface, unixTime int64, title, message, baseRef, headRef, headCommit string, draft bool, files []repository.Hash, metadata map[string]string) (*Bug, *CreateOperation, error) {
	b := NewBug()
	op := NewCreatePROp(author, unixTime, title, message, baseRef, headRef, headCommit, draft, files)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, op, err
	}
	b.Append(op)
	return b, op, nil
}
