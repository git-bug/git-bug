package bug

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag/dagtest"
	"github.com/git-bug/git-bug/repository"
)

func TestCreate(t *testing.T) {
	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	b, op, err := Create(rene, time.Now().Unix(), "title", "message", nil, nil)
	require.NoError(t, err)

	require.Equal(t, "title", op.Title)
	require.Equal(t, "message", op.Message)

	// Create generate the initial operation and create a new timeline item
	snap := b.Compile()
	require.Equal(t, common.OpenStatus, snap.Status)
	require.Equal(t, rene, snap.Author)
	require.Equal(t, "title", snap.Title)
	require.Len(t, snap.Operations, 1)
	require.Equal(t, op, snap.Operations[0])

	require.Len(t, snap.Timeline, 1)
	require.Equal(t, entity.CombineIds(b.Id(), op.Id()), snap.Timeline[0].CombinedId())
	require.Equal(t, rene, snap.Timeline[0].(*CreateTimelineItem).Author)
	require.Equal(t, "message", snap.Timeline[0].(*CreateTimelineItem).Message)
}

func TestCreateSerialize(t *testing.T) {
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*CreateOperation, entity.Resolvers) {
		return NewCreateOp(author, unixTime, "title", "message", nil), nil
	})
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*CreateOperation, entity.Resolvers) {
		return NewCreateOp(author, unixTime, "title", "message",
			[]repository.Hash{
				repository.Hash(strings.Repeat("a", 40)),
				repository.Hash(strings.Repeat("b", 40)),
			}), nil
	})
}

func TestCommitOperations(t *testing.T) {
	repo := repository.NewMockRepo()
	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	require.NoError(t, rene.Commit(repo))

	now := time.Now().Unix()
	create := func() *CreateOperation { return NewCreateOp(rene, now, "title", "message", nil) }
	comment := func() *AddCommentOperation { return NewAddCommentOp(rene, now, "comment", nil) }

	t.Run("a new bug starts with its Create op", func(t *testing.T) {
		b := NewBug()
		require.NoError(t, b.CommitOperations(repo, []Operation{create(), comment()}))

		read, err := Read(t.Context(), repo, b.Id())
		require.NoError(t, err)
		require.NoError(t, read.Validate())
		require.Len(t, read.Operations(), 2)
	})

	t.Run("a new bug without a Create op first panics", func(t *testing.T) {
		require.PanicsWithValue(t, "the first operation of a new bug should be a Create op", func() {
			_ = NewBug().CommitOperations(repo, []Operation{comment(), create()})
		})
	})

	t.Run("a second Create op panics", func(t *testing.T) {
		require.PanicsWithValue(t, "only one Create op allowed in a bug", func() {
			_ = NewBug().CommitOperations(repo, []Operation{create(), create()})
		})

		b := NewBug()
		require.NoError(t, b.CommitOperations(repo, []Operation{create()}))
		require.PanicsWithValue(t, "only one Create op allowed in a bug", func() {
			_ = b.CommitOperations(repo, []Operation{create()})
		})
	})
}
