package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/timestamp"
	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	snapshot := Snapshot{}

	rene := identity.NewBare("René Descartes", "rene@descartes.fr")
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
	var rene = identity.NewBare("René Descartes", "rene@descartes.fr")
	unix := time.Now().Unix()
	before := NewCreateOp(rene, unix, "title", "message", nil)

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after CreateOperation
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	// enforce creating the IDs
	before.Id()
	rene.Id()

	assert.Equal(t, before, &after)
}
