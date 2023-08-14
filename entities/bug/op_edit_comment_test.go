package bug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
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
}

func TestEditCommentSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author entity.Identity, unixTime int64) (*EditCommentOperation, entity.Resolvers) {
		return NewEditCommentOp(author, unixTime, "target", "message", nil), nil
	})
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author entity.Identity, unixTime int64) (*EditCommentOperation, entity.Resolvers) {
		return NewEditCommentOp(author, unixTime, "target", "message", []repository.Hash{"hash1", "hash2"}), nil
	})
}
