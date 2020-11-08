package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

func TestCreate(t *testing.T) {
	snapshot := Snapshot{}

	repo := repository.NewMockRepoClock()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	unix := time.Now().Unix()

	create := NewCreateOp(rene, unix, "title", "message", nil)

	create.Apply(&snapshot)

	id := create.Id()
	require.NoError(t, id.Validate())

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

	require.Equal(t, expected, snapshot)
}

func TestCreateSerialize(t *testing.T) {
	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	unix := time.Now().Unix()
	before := NewCreateOp(rene, unix, "title", "message", nil)

	data, err := json.Marshal(before)
	require.NoError(t, err)

	var after CreateOperation
	err = json.Unmarshal(data, &after)
	require.NoError(t, err)

	// enforce creating the ID
	before.Id()

	// Replace the identity stub with the real thing
	require.Equal(t, rene.Id(), after.base().Author.Id())
	after.Author = rene

	require.Equal(t, before, &after)
}
