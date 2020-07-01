package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreate(t *testing.T) {
	snapshot := Snapshot{}

	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	unix := time.Now().Unix()

	create := NewCreateOp(rene, unix, "title", "message", nil)

	create.Apply(&snapshot)

	id := create.Id()
	assert.NoError(t, id.Validate())

	comment := Comment{
		id:       id,
		Author:   rene,
		Message:  "message",
		UnixTime: timestamp.Timestamp(create.UnixTime),
	}

	expected := Snapshot{
		Title: "title",
		Comments: []Comment{
			comment,
		},
		Author:       rene,
		Participants: []identity.Interface{rene},
		Actors:       []identity.Interface{rene},
		CreateTime:   create.Time(),
		Timeline: []TimelineItem{
			&CreateTimelineItem{
				CommentTimelineItem: NewCommentTimelineItem(id, comment),
			},
		},
	}

	assert.Equal(t, expected, snapshot)
}

func TestCreateSerialize(t *testing.T) {
	repo := repository.NewMockRepoForTest()
	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	err := rene.Commit(repo)
	require.NoError(t, err)

	unix := time.Now().Unix()
	before := NewCreateOp(rene, unix, "title", "message", nil)

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after CreateOperation
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	// enforce creating the ID
	before.Id()

	// Replace the identity stub with the real thing
	assert.Equal(t, rene.Id(), after.base().Author.Id())
	after.Author = rene

	assert.Equal(t, before, &after)
}
