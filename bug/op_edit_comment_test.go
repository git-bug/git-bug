package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/stretchr/testify/assert"
)

func TestEdit(t *testing.T) {
	snapshot := Snapshot{}

	var rene = identity.NewIdentity("René Descartes", "rene@descartes.fr")

	unix := time.Now().Unix()

	create := NewCreateOp(rene, unix, "title", "create", nil)
	create.Apply(&snapshot)

	hash1, err := create.Hash()
	if err != nil {
		t.Fatal(err)
	}

	comment := NewAddCommentOp(rene, unix, "comment", nil)
	comment.Apply(&snapshot)

	hash2, err := comment.Hash()
	if err != nil {
		t.Fatal(err)
	}

	edit := NewEditCommentOp(rene, unix, hash1, "create edited", nil)
	edit.Apply(&snapshot)

	assert.Equal(t, len(snapshot.Timeline), 2)
	assert.Equal(t, len(snapshot.Timeline[0].(*CreateTimelineItem).History), 2)
	assert.Equal(t, len(snapshot.Timeline[1].(*AddCommentTimelineItem).History), 1)
	assert.Equal(t, snapshot.Comments[0].Message, "create edited")
	assert.Equal(t, snapshot.Comments[1].Message, "comment")

	edit2 := NewEditCommentOp(rene, unix, hash2, "comment edited", nil)
	edit2.Apply(&snapshot)

	assert.Equal(t, len(snapshot.Timeline), 2)
	assert.Equal(t, len(snapshot.Timeline[0].(*CreateTimelineItem).History), 2)
	assert.Equal(t, len(snapshot.Timeline[1].(*AddCommentTimelineItem).History), 2)
	assert.Equal(t, snapshot.Comments[0].Message, "create edited")
	assert.Equal(t, snapshot.Comments[1].Message, "comment edited")
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
