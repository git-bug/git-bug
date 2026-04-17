package bug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

func newTestPR(t *testing.T) (*Bug, identity.Interface) {
	t.Helper()
	repo := repository.NewMockRepo()
	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	b, _, err := CreatePR(rene, time.Now().Unix(), "t", "b",
		"refs/heads/main", "refs/heads/feat", "aaa111", false, nil, nil)
	require.NoError(t, err)
	return b, rene
}

func TestUpdateHead(t *testing.T) {
	b, rene := newTestPR(t)
	op, err := UpdateHead(b, rene, time.Now().Unix(), "bbb222", nil)
	require.NoError(t, err)
	require.Equal(t, "aaa111", op.PreviousCommit)
	require.Equal(t, "bbb222", op.NewCommit)

	snap := b.Compile()
	require.Equal(t, "bbb222", snap.HeadCommit)

	// Next update uses the most recent head as previous.
	op2, err := UpdateHead(b, rene, time.Now().Unix(), "ccc333", nil)
	require.NoError(t, err)
	require.Equal(t, "bbb222", op2.PreviousCommit)
}

func TestUpdateHeadRejectsIssue(t *testing.T) {
	repo := repository.NewMockRepo()
	rene, _ := identity.NewIdentity(repo, "R", "r@r")
	b, _, err := Create(rene, time.Now().Unix(), "t", "b", nil, nil)
	require.NoError(t, err)

	_, err = UpdateHead(b, rene, time.Now().Unix(), "xxx", nil)
	require.ErrorContains(t, err, "not a pull-request")
}

func TestAddReview(t *testing.T) {
	b, rene := newTestPR(t)
	id, op, err := AddReview(b, rene, time.Now().Unix(), ReviewApproved, "LGTM", "aaa111", nil)
	require.NoError(t, err)
	require.NotEqual(t, entity.UnsetCombinedId, id)
	require.Equal(t, ReviewApproved, op.State)

	snap := b.Compile()
	require.Len(t, snap.Reviews, 1)
	require.Equal(t, ReviewApproved, snap.Reviews[0].State)
	require.Equal(t, "aaa111", snap.Reviews[0].CommitHash)
}

func TestAddReviewComment(t *testing.T) {
	b, rene := newTestPR(t)
	reviewId, _, err := AddReview(b, rene, time.Now().Unix(), ReviewCommented, "", "aaa111", nil)
	require.NoError(t, err)

	commentId, _, err := AddReviewComment(b, rene, time.Now().Unix(), reviewId,
		"nit: typo", "aaa111", "foo.go", 10, 12, "", nil)
	require.NoError(t, err)
	require.NotEqual(t, entity.UnsetCombinedId, commentId)

	snap := b.Compile()
	require.Len(t, snap.Reviews, 1)
	require.Len(t, snap.Reviews[0].Comments, 1)
	require.Equal(t, "foo.go", snap.Reviews[0].Comments[0].Path)
	require.Equal(t, 10, snap.Reviews[0].Comments[0].StartLine)
	require.Equal(t, 12, snap.Reviews[0].Comments[0].EndLine)
}

func TestAddReviewCommentValidation(t *testing.T) {
	b, rene := newTestPR(t)
	reviewId, _, err := AddReview(b, rene, time.Now().Unix(), ReviewCommented, "", "aaa111", nil)
	require.NoError(t, err)

	_, _, err = AddReviewComment(b, rene, time.Now().Unix(), reviewId, "x", "aaa111", "foo.go", 0, 0, "", nil)
	require.ErrorContains(t, err, "start_line must be positive")

	_, _, err = AddReviewComment(b, rene, time.Now().Unix(), reviewId, "x", "aaa111", "foo.go", 10, 5, "", nil)
	require.ErrorContains(t, err, "precedes start_line")

	_, _, err = AddReviewComment(b, rene, time.Now().Unix(), "", "x", "aaa111", "foo.go", 1, 0, "", nil)
	require.ErrorContains(t, err, "review_id is empty")
}

func TestPROpsRejectIssue(t *testing.T) {
	repo := repository.NewMockRepo()
	rene, _ := identity.NewIdentity(repo, "R", "r@r")
	b, _, err := Create(rene, time.Now().Unix(), "t", "b", nil, nil)
	require.NoError(t, err)

	_, _, err = AddReview(b, rene, time.Now().Unix(), ReviewApproved, "", "abc", nil)
	require.ErrorContains(t, err, "not a pull-request")

	_, _, err = AddReviewComment(b, rene, time.Now().Unix(), "x", "body", "abc", "p", 1, 0, "", nil)
	require.ErrorContains(t, err, "not a pull-request")
}

func TestPROpsSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*UpdateHeadOperation, entity.Resolvers) {
		return NewUpdateHeadOp(author, unixTime, "new123", "old456"), nil
	})
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*AddReviewOperation, entity.Resolvers) {
		return NewAddReviewOp(author, unixTime, ReviewApproved, "LGTM", "commit1"), nil
	})
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*AddReviewCommentOperation, entity.Resolvers) {
		return NewAddReviewCommentOp(author, unixTime, "review-id", "body", "commit1", "foo.go", 10, 12, "reply-target"), nil
	})
}
