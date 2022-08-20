package bug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
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
		id:       entity.CombineIds(create.Id(), create.Id()),
		Author:   rene,
		Message:  "message",
		UnixTime: timestamp.Timestamp(create.UnixTime),
	}

	expected := Snapshot{
		id:    create.Id(),
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
				CommentTimelineItem: NewCommentTimelineItem(comment),
			},
		},
	}

	require.Equal(t, expected, snapshot)
}

func TestCreateSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, func(author identity.Interface, unixTime int64) *CreateOperation {
		return NewCreateOp(author, unixTime, "title", "message", nil)
	})
	dag.SerializeRoundTripTest(t, func(author identity.Interface, unixTime int64) *CreateOperation {
		return NewCreateOp(author, unixTime, "title", "message", []repository.Hash{"hash1", "hash2"})
	})
}
