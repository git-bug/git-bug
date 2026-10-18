package bug

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag/dagtest"
	"github.com/git-bug/git-bug/repository"
)

func TestEdit(t *testing.T) {
	snapshot := Snapshot{}

	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	unix := time.Now().Unix()

	create := NewCreateOp(rene, unix, "title", "create", nil)
	create.Apply(&snapshot)

	require.NoError(t, create.Id().Validate())

	comment1 := NewAddCommentOp(rene, unix, "comment 1", nil)
	comment1.Apply(&snapshot)

	require.NoError(t, comment1.Id().Validate())

	// add another unrelated op in between
	setTitle := NewSetTitleOp(rene, unix, "edited title", "title")
	setTitle.Apply(&snapshot)

	comment2 := NewAddCommentOp(rene, unix, "comment 2", nil)
	comment2.Apply(&snapshot)

	require.NoError(t, comment2.Id().Validate())

	edit := NewEditCommentOp(rene, unix, create.Id(), "create edited", nil)
	edit.Apply(&snapshot)

	require.Len(t, snapshot.Timeline, 4)
	require.Len(t, snapshot.Timeline[0].(*CreateTimelineItem).History, 2)
	require.Len(t, snapshot.Timeline[1].(*AddCommentTimelineItem).History, 1)
	require.Len(t, snapshot.Timeline[3].(*AddCommentTimelineItem).History, 1)
	require.Equal(t, snapshot.Comments[0].Message, "create edited")
	require.Equal(t, snapshot.Comments[1].Message, "comment 1")
	require.Equal(t, snapshot.Comments[2].Message, "comment 2")

	edit2 := NewEditCommentOp(rene, unix, comment1.Id(), "comment 1 edited", nil)
	edit2.Apply(&snapshot)

	require.Len(t, snapshot.Timeline, 4)
	require.Len(t, snapshot.Timeline[0].(*CreateTimelineItem).History, 2)
	require.Len(t, snapshot.Timeline[1].(*AddCommentTimelineItem).History, 2)
	require.Len(t, snapshot.Timeline[3].(*AddCommentTimelineItem).History, 1)
	require.Equal(t, snapshot.Comments[0].Message, "create edited")
	require.Equal(t, snapshot.Comments[1].Message, "comment 1 edited")
	require.Equal(t, snapshot.Comments[2].Message, "comment 2")

	edit3 := NewEditCommentOp(rene, unix, comment2.Id(), "comment 2 edited", nil)
	edit3.Apply(&snapshot)

	require.Len(t, snapshot.Timeline, 4)
	require.Len(t, snapshot.Timeline[0].(*CreateTimelineItem).History, 2)
	require.Len(t, snapshot.Timeline[1].(*AddCommentTimelineItem).History, 2)
	require.Len(t, snapshot.Timeline[3].(*AddCommentTimelineItem).History, 2)
	require.Equal(t, snapshot.Comments[0].Message, "create edited")
	require.Equal(t, snapshot.Comments[1].Message, "comment 1 edited")
	require.Equal(t, snapshot.Comments[2].Message, "comment 2 edited")

	t.Run("history records who wrote each step", func(t *testing.T) {
		isaac, err := identity.NewIdentity(repo, "Isaac Newton", "isaac@newton.uk")
		require.NoError(t, err)

		edit4 := NewEditCommentOp(isaac, unix, comment2.Id(), "comment 2 edited by isaac", nil)
		edit4.Apply(&snapshot)

		history := snapshot.Timeline[3].(*AddCommentTimelineItem).History
		require.Len(t, history, 3)
		require.Equal(t, rene.Id(), history[0].Author.Id())
		require.Equal(t, rene.Id(), history[1].Author.Id())
		require.Equal(t, isaac.Id(), history[2].Author.Id())
		// the comment keeps its original author
		require.Equal(t, rene.Id(), snapshot.Timeline[3].(*AddCommentTimelineItem).Author.Id())
	})
}

func TestEditCommentSerialize(t *testing.T) {
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*EditCommentOperation, entity.Resolvers) {
		return NewEditCommentOp(author, unixTime, entity.DeriveId([]byte("target")), "message", nil), nil
	})
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*EditCommentOperation, entity.Resolvers) {
		return NewEditCommentOp(author, unixTime, entity.DeriveId([]byte("target")),
			"message", []repository.Hash{
				repository.Hash(strings.Repeat("a", 40)),
				repository.Hash(strings.Repeat("b", 40)),
			}), nil
	})
}
