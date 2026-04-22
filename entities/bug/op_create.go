package bug

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/text"
	"github.com/git-bug/git-bug/util/timestamp"
)

// SHA-1 (40) or SHA-256 (64) lowercase hex. Accepts either so the format
// is future-proof for git's SHA-256 transition.
var gitHashRe = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

// validBranchRef reports whether s is a full ref path of the form
// "refs/heads/<name>" where <name> obeys git's check-ref-format rules.
// Used to gate BaseRef / HeadRef on PR ops so attacker-controlled branch
// names can't later be passed to git as crafted refs (e.g. ".." traversal,
// "@{" reflog syntax, refs/bugs collisions).
func validBranchRef(s string) bool {
	const prefix = "refs/heads/"
	if !strings.HasPrefix(s, prefix) {
		return false
	}
	name := s[len(prefix):]
	if name == "" || name == "@" {
		return false
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") ||
		strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") ||
		strings.HasSuffix(name, ".lock") {
		return false
	}
	if strings.Contains(name, "..") || strings.Contains(name, "//") ||
		strings.Contains(name, "@{") {
		return false
	}
	// Each slash-separated component must not start with a dot.
	for _, comp := range strings.Split(name, "/") {
		if strings.HasPrefix(comp, ".") {
			return false
		}
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return false
		}
		switch r {
		case ' ', '~', '^', ':', '?', '*', '[', '\\', '\x7f':
			return false
		}
	}
	return true
}

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
		if !validBranchRef(op.BaseRef) {
			return fmt.Errorf("pr base_ref must be a valid refs/heads/<name>")
		}
		if !validBranchRef(op.HeadRef) {
			return fmt.Errorf("pr head_ref must be a valid refs/heads/<name>")
		}
		if op.HeadCommit != "" && !gitHashRe.MatchString(op.HeadCommit) {
			return fmt.Errorf("pr head_commit is not a lowercase hex git hash")
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
