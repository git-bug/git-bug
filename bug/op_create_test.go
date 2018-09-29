package bug

import (
	"reflect"
	"testing"
	"time"
)

func TestCreate(t *testing.T) {
	snapshot := Snapshot{}

	var rene = Person{
		Name:  "René Descartes",
		Email: "rene@descartes.fr",
	}

	unix := time.Now().Unix()

	create := NewCreateOp(rene, unix, "title", "message", nil)

	create.Apply(&snapshot)

	expected := Snapshot{
		Title: "title",
		Comments: []Comment{
			{Author: rene, Message: "message", UnixTime: create.UnixTime},
		},
		Author:    rene,
		CreatedAt: create.Time(),
	}

	if !reflect.DeepEqual(snapshot, expected) {
		t.Fatalf("%v different than %v", snapshot, expected)
	}
}
