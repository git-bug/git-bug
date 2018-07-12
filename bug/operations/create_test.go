package operations

import (
	"github.com/MichaelMure/git-bug/bug"
	"reflect"
	"testing"
)

func TestCreate(t *testing.T) {
	snapshot := bug.Snapshot{}

	var rene = bug.Person{
		Name:  "René Descartes",
		Email: "rene@descartes.fr",
	}

	create := NewCreateOp(rene, "title", "message")

	snapshot = create.Apply(snapshot)

	expected := bug.Snapshot{
		Title: "title",
		Comments: []bug.Comment{
			{Author: rene, Message: "message"},
		},
	}

	if !reflect.DeepEqual(snapshot, expected) {
		t.Fail()
	}
}
