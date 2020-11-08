package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/identity"
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

	id1 := create.Id()
	require.NoError(t, id1.Validate())

	comment1 := NewAddCommentOp(rene, unix, "comment 1", nil)
	comment1.Apply(&snapshot)

	id2 := comment1.Id()
	require.NoError(t, id2.Validate())

	// add another unrelated op in between
	setTitle := NewSetTitleOp(rene, unix, "edited title", "title")
	setTitle.Apply(&snapshot)

	comment2 := NewAddCommentOp(rene, unix, "comment 2", nil)
	comment2.Apply(&snapshot)

	id3 := comment2.Id()
	require.NoError(t, id3.Validate())

	edit := NewEditCommentOp(rene, unix, id1, "create edited", nil)
	edit.Apply(&snapshot)

	require.Equal(t, len(snapshot.Timeline), 4)
	require.Equal(t, len(snapshot.Timeline[0].(*CreateTimelineItem).History), 2)
	require.Equal(t, len(snapshot.Timeline[1].(*AddCommentTimelineItem).History), 1)
	require.Equal(t, len(snapshot.Timeline[3].(*AddCommentTimelineItem).History), 1)
	require.Equal(t, snapshot.Comments[0].Message, "create edited")
	require.Equal(t, snapshot.Comments[1].Message, "comment 1")
	require.Equal(t, snapshot.Comments[2].Message, "comment 2")

	edit2 := NewEditCommentOp(rene, unix, id2, "comment 1 edited", nil)
	edit2.Apply(&snapshot)

	require.Equal(t, len(snapshot.Timeline), 4)
	require.Equal(t, len(snapshot.Timeline[0].(*CreateTimelineItem).History), 2)
	require.Equal(t, len(snapshot.Timeline[1].(*AddCommentTimelineItem).History), 2)
	require.Equal(t, len(snapshot.Timeline[3].(*AddCommentTimelineItem).History), 1)
	require.Equal(t, snapshot.Comments[0].Message, "create edited")
	require.Equal(t, snapshot.Comments[1].Message, "comment 1 edited")
	require.Equal(t, snapshot.Comments[2].Message, "comment 2")

	edit3 := NewEditCommentOp(rene, unix, id3, "comment 2 edited", nil)
	edit3.Apply(&snapshot)

	require.Equal(t, len(snapshot.Timeline), 4)
	require.Equal(t, len(snapshot.Timeline[0].(*CreateTimelineItem).History), 2)
	require.Equal(t, len(snapshot.Timeline[1].(*AddCommentTimelineItem).History), 2)
	require.Equal(t, len(snapshot.Timeline[3].(*AddCommentTimelineItem).History), 2)
	require.Equal(t, snapshot.Comments[0].Message, "create edited")
	require.Equal(t, snapshot.Comments[1].Message, "comment 1 edited")
	require.Equal(t, snapshot.Comments[2].Message, "comment 2 edited")
}

func TestEditCommentSerialize(t *testing.T) {
	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	unix := time.Now().Unix()
	before := NewEditCommentOp(rene, unix, "target", "message", nil)

	data, err := json.Marshal(before)
	require.NoError(t, err)

	var after EditCommentOperation
	err = json.Unmarshal(data, &after)
	require.NoError(t, err)

	// enforce creating the ID
	before.Id()

	// Replace the identity stub with the real thing
	require.Equal(t, rene.Id(), after.base().Author.Id())
	after.Author = rene

	require.Equal(t, before, &after)
}
