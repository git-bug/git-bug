package bug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

func TestCreatePR(t *testing.T) {
	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	b, op, err := CreatePR(rene, time.Now().Unix(), "new feature", "body",
		"refs/heads/main", "refs/heads/feat", "0000000000000000000000000000000000000010", false, nil, nil)
	require.NoError(t, err)
	require.Equal(t, common.PRKind, op.Kind)

	snap := b.Compile()
	require.Equal(t, common.PRKind, snap.Kind)
	require.Equal(t, common.OpenStatus, snap.Status)
	require.Equal(t, "refs/heads/main", snap.BaseRef)
	require.Equal(t, "refs/heads/feat", snap.HeadRef)
	require.Equal(t, "0000000000000000000000000000000000000010", snap.HeadCommit)
}

func TestCreatePRDraft(t *testing.T) {
	repo := repository.NewMockRepo()
	rene, _ := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")

	b, _, err := CreatePR(rene, time.Now().Unix(), "wip", "body",
		"refs/heads/main", "refs/heads/wip", "0000000000000000000000000000000000000011", true, nil, nil)
	require.NoError(t, err)

	snap := b.Compile()
	require.Equal(t, common.DraftStatus, snap.Status)
}

func TestCreatePRRequiresRefs(t *testing.T) {
	repo := repository.NewMockRepo()
	rene, _ := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")

	hash := "0000000000000000000000000000000000000020"
	_, _, err := CreatePR(rene, time.Now().Unix(), "t", "b", "", "refs/heads/feat", hash, false, nil, nil)
	require.ErrorContains(t, err, "base_ref")

	_, _, err = CreatePR(rene, time.Now().Unix(), "t", "b", "refs/heads/main", "", hash, false, nil, nil)
	require.ErrorContains(t, err, "head_ref")
}

func TestCreateIssueRejectsPRFields(t *testing.T) {
	repo := repository.NewMockRepo()
	rene, _ := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")

	op := NewCreateOp(rene, time.Now().Unix(), "t", "b", nil)
	op.BaseRef = "refs/heads/main"
	require.ErrorContains(t, op.Validate(), "issue must not carry PR fields")
}

func TestCreatePRSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*CreateOperation, entity.Resolvers) {
		return NewCreatePROp(author, unixTime, "title", "body",
			"refs/heads/main", "refs/heads/feat", "0000000000000000000000000000000000000010", false, nil), nil
	})
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*CreateOperation, entity.Resolvers) {
		return NewCreatePROp(author, unixTime, "title", "body",
			"refs/heads/main", "refs/heads/draft", "0000000000000000000000000000000000000011", true, nil), nil
	})
}

func TestMergeOp(t *testing.T) {
	repo := repository.NewMockRepo()
	rene, _ := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")

	b, _, err := CreatePR(rene, time.Now().Unix(), "t", "b",
		"refs/heads/main", "refs/heads/feat",
		"0000000000000000000000000000000000000021", false, nil, nil)
	require.NoError(t, err)

	merge := "0000000000000000000000000000000000000022"
	_, err = Merge(b, rene, time.Now().Unix(), merge, nil)
	require.NoError(t, err)

	snap := b.Compile()
	require.Equal(t, common.MergedStatus, snap.Status)
	require.Equal(t, merge, snap.MergeCommit)
}

func TestMergeRequiresCommit(t *testing.T) {
	repo := repository.NewMockRepo()
	rene, _ := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")

	op := NewMergeOp(rene, time.Now().Unix(), "")
	require.ErrorContains(t, op.Validate(), "merge requires merge_commit")
}

func TestSetStatusRejectsMergeCommitUnlessMerged(t *testing.T) {
	repo := repository.NewMockRepo()
	rene, _ := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")

	op := NewSetStatusOp(rene, time.Now().Unix(), common.ClosedStatus)
	op.MergeCommit = "bogus"
	require.ErrorContains(t, op.Validate(), "merge_commit only valid when status=merged")
}

func TestExistingIssueDeserializesAsIssue(t *testing.T) {
	// Zero-value Kind means existing serialized bugs (written before PR
	// support landed) still deserialize correctly as issues.
	var op CreateOperation
	require.Equal(t, common.IssueKind, op.Kind)
}

// Entity wrapper types used by dag package must implement entity.Resolved.
var _ entity.Resolved = (*Bug)(nil)
