package repository

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

// ChangeStatus describes how a file was affected by a commit.
type ChangeStatus string

const (
	ChangeStatusAdded    ChangeStatus = "added"
	ChangeStatusModified ChangeStatus = "modified"
	ChangeStatusDeleted  ChangeStatus = "deleted"
	ChangeStatusRenamed  ChangeStatus = "renamed"
)

func (s ChangeStatus) MarshalGQL(w io.Writer) {
	switch s {
	case ChangeStatusAdded:
		fmt.Fprint(w, strconv.Quote("ADDED"))
	case ChangeStatusModified:
		fmt.Fprint(w, strconv.Quote("MODIFIED"))
	case ChangeStatusDeleted:
		fmt.Fprint(w, strconv.Quote("DELETED"))
	case ChangeStatusRenamed:
		fmt.Fprint(w, strconv.Quote("RENAMED"))
	default:
		panic(fmt.Sprintf("unknown ChangeStatus value %q", string(s)))
	}
}

func (s *ChangeStatus) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}
	switch str {
	case "ADDED":
		*s = ChangeStatusAdded
	case "MODIFIED":
		*s = ChangeStatusModified
	case "DELETED":
		*s = ChangeStatusDeleted
	case "RENAMED":
		*s = ChangeStatusRenamed
	default:
		return fmt.Errorf("%q is not a valid ChangeStatus", str)
	}
	return nil
}

// DiffLineType is the role of a line within a unified diff hunk.
type DiffLineType string

const (
	DiffLineContext DiffLineType = "context"
	DiffLineAdded   DiffLineType = "added"
	DiffLineDeleted DiffLineType = "deleted"
)

func (t DiffLineType) MarshalGQL(w io.Writer) {
	switch t {
	case DiffLineContext:
		fmt.Fprint(w, strconv.Quote("CONTEXT"))
	case DiffLineAdded:
		fmt.Fprint(w, strconv.Quote("ADDED"))
	case DiffLineDeleted:
		fmt.Fprint(w, strconv.Quote("DELETED"))
	default:
		panic(fmt.Sprintf("unknown DiffLineType value %q", string(t)))
	}
}

func (t *DiffLineType) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}
	switch str {
	case "CONTEXT":
		*t = DiffLineContext
	case "ADDED":
		*t = DiffLineAdded
	case "DELETED":
		*t = DiffLineDeleted
	default:
		return fmt.Errorf("%q is not a valid DiffLineType", str)
	}
	return nil
}

// CommitMeta holds the metadata for a single commit, suitable for listing.
type CommitMeta struct {
	Hash        Hash
	Message     string
	AuthorName  string
	AuthorEmail string
	Date        time.Time
	Parents     []Hash
}

// ChangedFile describes a file that was modified in a commit.
type ChangedFile struct {
	Path    string
	OldPath *string // non-nil for renames
	Status  ChangeStatus
}

// CommitDetail extends CommitMeta with the full message and the list of
// changed files (relative to the first parent).
type CommitDetail struct {
	CommitMeta
	FullMessage string
	Files       []ChangedFile
}

// DiffLine represents one line in a unified diff hunk.
type DiffLine struct {
	Type    DiffLineType
	Content string
	OldLine int
	NewLine int
}

// DiffHunk is a contiguous block of changes in a unified diff.
type DiffHunk struct {
	OldStart int
	OldLines int
	NewStart int
	NewLines int
	Lines    []DiffLine
}

// FileDiff is the diff for a single file in a commit.
type FileDiff struct {
	Path     string
	OldPath  *string // non-nil for renames
	IsBinary bool
	IsNew    bool
	IsDelete bool
	Hunks    []DiffHunk
}

// BranchInfo describes a local branch returned by RepoBrowse.Branches.
type BranchInfo struct {
	Name string
	Hash Hash // commit hash
}

// TagInfo describes a tag returned by RepoBrowse.Tags.
type TagInfo struct {
	Name string
	// Hash is always the target commit hash.  For annotated tags the tag
	// object is dereferenced; for lightweight tags this is the ref hash.
	Hash Hash
}
