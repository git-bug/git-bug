package bug

import (
	"fmt"
	"strings"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/text"
	"github.com/git-bug/git-bug/util/timestamp"
)

var _ Operation = &AddReviewCommentOperation{}

// AddReviewCommentOperation adds a line-anchored comment to a PR review.
//
// Anchors (CommitHash + Path + StartLine..EndLine) are immutable: later pushes
// to the branch do not move or invalidate these comments. When ReplyTo is set,
// this comment is a reply within a review thread rather than a new root.
// Valid only when the bug's Kind is PRKind.
type AddReviewCommentOperation struct {
	dag.OpBase
	// ReviewId is the combined id of the AddReviewOperation this comment attaches to.
	ReviewId   entity.CombinedId `json:"review_id"`
	Body       string            `json:"body"`
	CommitHash string            `json:"commit_hash"`
	Path       string            `json:"path"`
	StartLine  int               `json:"start_line"`
	EndLine    int               `json:"end_line,omitempty"`
	// ReplyTo is the combined id of the ReviewComment this replies to, if any.
	ReplyTo entity.CombinedId `json:"reply_to,omitempty"`
}

func (op *AddReviewCommentOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *AddReviewCommentOperation) Apply(snapshot *Snapshot) {
	if snapshot.Kind != common.PRKind {
		return
	}
	snapshot.addActor(op.Author())
	snapshot.addParticipant(op.Author())

	combinedId := entity.CombineIds(snapshot.Id(), op.Id())
	rc := ReviewComment{
		combinedId: combinedId,
		Author:     op.Author(),
		Body:       op.Body,
		CommitHash: op.CommitHash,
		Path:       op.Path,
		StartLine:  op.StartLine,
		EndLine:    op.EndLine,
		ReplyTo:    op.ReplyTo,
		CreatedAt:  timestamp.Timestamp(op.UnixTime),
	}

	// Attach to the matching Review if present; otherwise append to Timeline only.
	for i := range snapshot.Reviews {
		if snapshot.Reviews[i].combinedId == op.ReviewId {
			snapshot.Reviews[i].Comments = append(snapshot.Reviews[i].Comments, rc)
			break
		}
	}

	item := &AddReviewCommentTimelineItem{
		combinedId: combinedId,
		Author:     op.Author(),
		UnixTime:   rc.CreatedAt,
		ReviewId:   op.ReviewId,
		Body:       op.Body,
		CommitHash: op.CommitHash,
		Path:       op.Path,
		StartLine:  op.StartLine,
		EndLine:    op.EndLine,
		ReplyTo:    op.ReplyTo,
	}
	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *AddReviewCommentOperation) Validate() error {
	if err := op.OpBase.Validate(op, AddReviewCommentOp); err != nil {
		return err
	}
	if op.ReviewId == "" {
		return fmt.Errorf("review_id is empty")
	}
	if text.Empty(op.Body) {
		return fmt.Errorf("body is empty")
	}
	if !text.Safe(op.Body) {
		return fmt.Errorf("body is not fully printable")
	}
	if text.Empty(op.CommitHash) {
		return fmt.Errorf("commit_hash is empty")
	}
	if !gitHashRe.MatchString(op.CommitHash) {
		return fmt.Errorf("commit_hash is not a lowercase hex git hash")
	}
	if text.Empty(op.Path) {
		return fmt.Errorf("path is empty")
	}
	if !text.SafeOneLine(op.Path) {
		return fmt.Errorf("path has unsafe characters")
	}
	// Defence in depth: file paths from external sources should never escape
	// the repo root. Reject absolute paths, Windows-drive prefixes, any ..
	// segment, and backslashes (Windows separators or git-quoted bytes).
	if strings.HasPrefix(op.Path, "/") ||
		strings.Contains(op.Path, "\\") ||
		strings.Contains(op.Path, "..") ||
		(len(op.Path) >= 2 && op.Path[1] == ':') {
		return fmt.Errorf("path must be a repo-relative POSIX path")
	}
	if op.StartLine <= 0 {
		return fmt.Errorf("start_line must be positive")
	}
	if op.EndLine != 0 && op.EndLine < op.StartLine {
		return fmt.Errorf("end_line (%d) precedes start_line (%d)", op.EndLine, op.StartLine)
	}
	return nil
}

func NewAddReviewCommentOp(author identity.Interface, unixTime int64, reviewId entity.CombinedId, body, commitHash, path string, startLine, endLine int, replyTo entity.CombinedId) *AddReviewCommentOperation {
	return &AddReviewCommentOperation{
		OpBase:     dag.NewOpBase(AddReviewCommentOp, author, unixTime),
		ReviewId:   reviewId,
		Body:       body,
		CommitHash: commitHash,
		Path:       path,
		StartLine:  startLine,
		EndLine:    endLine,
		ReplyTo:    replyTo,
	}
}

type AddReviewCommentTimelineItem struct {
	combinedId entity.CombinedId
	Author     identity.Interface
	UnixTime   timestamp.Timestamp
	ReviewId   entity.CombinedId
	Body       string
	CommitHash string
	Path       string
	StartLine  int
	EndLine    int
	ReplyTo    entity.CombinedId
}

func (a *AddReviewCommentTimelineItem) CombinedId() entity.CombinedId { return a.combinedId }
func (a *AddReviewCommentTimelineItem) IsAuthored()                   {}

// AddReviewComment appends a line-anchored review comment to a PR.
func AddReviewComment(b Interface, author identity.Interface, unixTime int64, reviewId entity.CombinedId, body, commitHash, path string, startLine, endLine int, replyTo entity.CombinedId, metadata map[string]string) (entity.CombinedId, *AddReviewCommentOperation, error) {
	create, ok := b.FirstOp().(*CreateOperation)
	if !ok || create.Kind != common.PRKind {
		return entity.UnsetCombinedId, nil, fmt.Errorf("AddReviewComment: bug is not a pull-request")
	}

	op := NewAddReviewCommentOp(author, unixTime, reviewId, body, commitHash, path, startLine, endLine, replyTo)
	for k, v := range metadata {
		op.SetMetadata(k, v)
	}
	if err := op.Validate(); err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	b.Append(op)
	return entity.CombineIds(b.Id(), op.Id()), op, nil
}
