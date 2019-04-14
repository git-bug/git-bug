package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/identity"
)

func TestEdit(t *testing.T) {
	snapshot := Snapshot{}

	rene := identity.NewBare("René Descartes", "rene@descartes.fr")
	unix := time.Now().Unix()

	create := NewCreateOp(rene, unix, "title", "create", nil)
	create.Apply(&snapshot)

	hash1, err := create.Hash()
	require.NoError(t, err)

	comment1 := NewAddCommentOp(rene, unix, "comment 1", nil)
	comment1.Apply(&snapshot)

	hash2, err := comment1.Hash()
	require.NoError(t, err)

	// add another unrelated op in between
	setTitle := NewSetTitleOp(rene, unix, "edited title", "title")
	setTitle.Apply(&snapshot)

	comment2 := NewAddCommentOp(rene, unix, "comment 2", nil)
	comment2.Apply(&snapshot)

	hash3, err := comment2.Hash()
	require.NoError(t, err)

	edit := NewEditCommentOp(rene, unix, hash1, "create edited", nil)
	edit.Apply(&snapshot)

	assert.Equal(t, len(snapshot.Timeline), 4)
	assert.Equal(t, len(snapshot.Timeline[0].(*CreateTimelineItem).History), 2)
	assert.Equal(t, len(snapshot.Timeline[1].(*AddCommentTimelineItem).History), 1)
	assert.Equal(t, len(snapshot.Timeline[3].(*AddCommentTimelineItem).History), 1)
	assert.Equal(t, snapshot.Comments[0].Message, "create edited")
	assert.Equal(t, snapshot.Comments[1].Message, "comment 1")
	assert.Equal(t, snapshot.Comments[2].Message, "comment 2")

	edit2 := NewEditCommentOp(rene, unix, hash2, "comment 1 edited", nil)
	edit2.Apply(&snapshot)

	assert.Equal(t, len(snapshot.Timeline), 4)
	assert.Equal(t, len(snapshot.Timeline[0].(*CreateTimelineItem).History), 2)
	assert.Equal(t, len(snapshot.Timeline[1].(*AddCommentTimelineItem).History), 2)
	assert.Equal(t, len(snapshot.Timeline[3].(*AddCommentTimelineItem).History), 1)
	assert.Equal(t, snapshot.Comments[0].Message, "create edited")
	assert.Equal(t, snapshot.Comments[1].Message, "comment 1 edited")
	assert.Equal(t, snapshot.Comments[2].Message, "comment 2")

	edit3 := NewEditCommentOp(rene, unix, hash3, "comment 2 edited", nil)
	edit3.Apply(&snapshot)

	assert.Equal(t, len(snapshot.Timeline), 4)
	assert.Equal(t, len(snapshot.Timeline[0].(*CreateTimelineItem).History), 2)
	assert.Equal(t, len(snapshot.Timeline[1].(*AddCommentTimelineItem).History), 2)
	assert.Equal(t, len(snapshot.Timeline[3].(*AddCommentTimelineItem).History), 2)
	assert.Equal(t, snapshot.Comments[0].Message, "create edited")
	assert.Equal(t, snapshot.Comments[1].Message, "comment 1 edited")
	assert.Equal(t, snapshot.Comments[2].Message, "comment 2 edited")
}

func TestEditCommentSerialize(t *testing.T) {
	var rene = identity.NewBare("René Descartes", "rene@descartes.fr")
	unix := time.Now().Unix()
	before := NewEditCommentOp(rene, unix, "target", "message", nil)

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after EditCommentOperation
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	assert.Equal(t, before, &after)
}
