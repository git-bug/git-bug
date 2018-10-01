package bug

import (
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/util/git"
)

func TestValidate(t *testing.T) {
	rene := Person{
		Name:  "René Descartes",
		Email: "rene@descartes.fr",
	}

	unix := time.Now().Unix()

	good := []Operation{
		NewCreateOp(rene, unix, "title", "message", nil),
		NewSetTitleOp(rene, unix, "title2", "title1"),
		NewAddCommentOp(rene, unix, "message2", nil),
		NewSetStatusOp(rene, unix, ClosedStatus),
		NewLabelChangeOperation(rene, unix, []Label{"added"}, []Label{"removed"}),
	}

	for _, op := range good {
		if err := op.Validate(); err != nil {
			t.Fatal(err)
		}
	}

	bad := []Operation{
		// opbase
		NewSetStatusOp(Person{Name: "", Email: "rene@descartes.fr"}, unix, ClosedStatus),
		NewSetStatusOp(Person{Name: "René Descartes\u001b", Email: "rene@descartes.fr"}, unix, ClosedStatus),
		NewSetStatusOp(Person{Name: "René Descartes", Email: "rene@descartes.fr\u001b"}, unix, ClosedStatus),
		NewSetStatusOp(Person{Name: "René \nDescartes", Email: "rene@descartes.fr"}, unix, ClosedStatus),
		NewSetStatusOp(Person{Name: "René Descartes", Email: "rene@\ndescartes.fr"}, unix, ClosedStatus),
		&CreateOperation{OpBase: OpBase{
			Author:        rene,
			UnixTime:      0,
			OperationType: CreateOp,
		},
			Title:   "title",
			Message: "message",
		},

		NewCreateOp(rene, unix, "multi\nline", "message", nil),
		NewCreateOp(rene, unix, "title", "message", []git.Hash{git.Hash("invalid")}),
		NewCreateOp(rene, unix, "title\u001b", "message", nil),
		NewCreateOp(rene, unix, "title", "message\u001b", nil),
		NewSetTitleOp(rene, unix, "multi\nline", "title1"),
		NewSetTitleOp(rene, unix, "title", "multi\nline"),
		NewSetTitleOp(rene, unix, "title\u001b", "title2"),
		NewSetTitleOp(rene, unix, "title", "title2\u001b"),
		NewAddCommentOp(rene, unix, "", nil),
		NewAddCommentOp(rene, unix, "message\u001b", nil),
		NewAddCommentOp(rene, unix, "message", []git.Hash{git.Hash("invalid")}),
		NewSetStatusOp(rene, unix, 1000),
		NewSetStatusOp(rene, unix, 0),
		NewLabelChangeOperation(rene, unix, []Label{}, []Label{}),
		NewLabelChangeOperation(rene, unix, []Label{"multi\nline"}, []Label{}),
	}

	for i, op := range bad {
		if err := op.Validate(); err == nil {
			t.Fatal("validation should have failed", i, op)
		}
	}
}
