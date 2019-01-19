package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	snapshot := Snapshot{}

	var rene = identity.NewIdentity("René Descartes", "rene@descartes.fr")

	unix := time.Now().Unix()

	create := NewCreateOp(rene, unix, "title", "message", nil)

	create.Apply(&snapshot)

	hash, err := create.Hash()
	if err != nil {
		t.Fatal(err)
	}

	comment := Comment{Author: rene, Message: "message", UnixTime: Timestamp(create.UnixTime)}

	expected := Snapshot{
		Title: "title",
		Comments: []Comment{
			comment,
		},
		Author:    rene,
		CreatedAt: create.Time(),
		Timeline: []TimelineItem{
			&CreateTimelineItem{
				CommentTimelineItem: NewCommentTimelineItem(hash, comment),
			},
		},
	}

	deep.CompareUnexportedFields = true
	if diff := deep.Equal(snapshot, expected); diff != nil {
		t.Fatal(diff)
	}
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

	assert.Equal(t, before, &after)
}
