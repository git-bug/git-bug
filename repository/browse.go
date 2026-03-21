package repository

import "time"

// ChangeStatus describes how a file was affected by a commit.
type ChangeStatus string

const (
	ChangeStatusAdded    ChangeStatus = "added"
	ChangeStatusModified ChangeStatus = "modified"
	ChangeStatusDeleted  ChangeStatus = "deleted"
	ChangeStatusRenamed  ChangeStatus = "renamed"
)

// DiffLineType is the role of a line within a unified diff hunk.
type DiffLineType string

const (
	DiffLineContext DiffLineType = "context"
	DiffLineAdded   DiffLineType = "added"
	DiffLineDeleted DiffLineType = "deleted"
)

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
	OldPath string // non-empty for renames
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
	OldPath  string
	IsBinary bool
	IsNew    bool
	IsDelete bool
	Hunks    []DiffHunk
}

// BranchInfo describes a local branch returned by RepoBrowse.Branches.
type BranchInfo struct {
	Name      string
	Hash      Hash // commit hash
	IsDefault bool // true for the branch HEAD points to
}

// TagInfo describes a tag returned by RepoBrowse.Tags.
type TagInfo struct {
	Name string
	// Hash is always the target commit hash.  For annotated tags the tag
	// object is dereferenced; for lightweight tags this is the ref hash.
	Hash Hash
}
