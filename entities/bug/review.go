package bug

import (
	"fmt"
	"io"
	"strconv"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/util/timestamp"
)

// ReviewState is the verdict attached to a PR review.
type ReviewState int

const (
	_ ReviewState = iota
	ReviewCommented
	ReviewApproved
	ReviewChangesRequested
)

func (r ReviewState) String() string {
	switch r {
	case ReviewCommented:
		return "commented"
	case ReviewApproved:
		return "approved"
	case ReviewChangesRequested:
		return "changes-requested"
	default:
		return "unknown review state"
	}
}

func (r ReviewState) Validate() error {
	switch r {
	case ReviewCommented, ReviewApproved, ReviewChangesRequested:
		return nil
	default:
		return fmt.Errorf("invalid review state")
	}
}

func (r ReviewState) MarshalGQL(w io.Writer) {
	switch r {
	case ReviewCommented:
		_, _ = w.Write([]byte(strconv.Quote("COMMENTED")))
	case ReviewApproved:
		_, _ = w.Write([]byte(strconv.Quote("APPROVED")))
	case ReviewChangesRequested:
		_, _ = w.Write([]byte(strconv.Quote("CHANGES_REQUESTED")))
	default:
		panic("missing case")
	}
}

func (r *ReviewState) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}
	switch str {
	case "COMMENTED":
		*r = ReviewCommented
	case "APPROVED":
		*r = ReviewApproved
	case "CHANGES_REQUESTED":
		*r = ReviewChangesRequested
	default:
		return fmt.Errorf("%s is not a valid ReviewState", str)
	}
	return nil
}

// Review is a compiled view of a PR review assembled from an AddReviewOperation
// and any AddReviewCommentOperations that reference it.
type Review struct {
	// combinedId is the snapshot-id combined with the AddReviewOperation id
	combinedId entity.CombinedId
	Author     identity.Interface
	State      ReviewState
	Body       string
	// CommitHash is the head commit the review was made against.
	CommitHash string
	CreatedAt  timestamp.Timestamp
	Comments   []ReviewComment
}

func (r Review) CombinedId() entity.CombinedId {
	return r.combinedId
}

// IsAuthored is a sign-post method for gqlgen.
func (r Review) IsAuthored() {}

// ReviewComment is a line-anchored comment attached to a Review. Anchors are
// immutable (commit hash + path + line range), so comments don't follow later
// pushes to the branch.
type ReviewComment struct {
	combinedId entity.CombinedId
	Author     identity.Interface
	Body       string
	CommitHash string
	Path       string
	StartLine  int
	EndLine    int
	// ReplyTo, when non-empty, identifies the ReviewComment this one replies to.
	ReplyTo   entity.CombinedId
	CreatedAt timestamp.Timestamp
}

func (rc ReviewComment) CombinedId() entity.CombinedId {
	return rc.combinedId
}

// IsAuthored is a sign-post method for gqlgen.
func (rc ReviewComment) IsAuthored() {}
