package operations

import (
	"testing"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/util/git"
)

func TestValidate(t *testing.T) {
	rene := bug.Person{
		Name:  "René Descartes",
		Email: "rene@descartes.fr",
	}

	good := []bug.Operation{
		NewCreateOp(rene, "title", "message", nil),
		NewSetTitleOp(rene, "title2", "title1"),
		NewAddCommentOp(rene, "message2", nil),
		NewSetStatusOp(rene, bug.ClosedStatus),
		NewLabelChangeOperation(rene, []bug.Label{"added"}, []bug.Label{"removed"}),
	}

	for _, op := range good {
		if err := op.Validate(); err != nil {
			t.Fatal(err)
		}
	}

	bad := []bug.Operation{
		// opbase
		NewSetStatusOp(bug.Person{Name: "", Email: "rene@descartes.fr"}, bug.ClosedStatus),
		NewSetStatusOp(bug.Person{Name: "René Descartes\u001b", Email: "rene@descartes.fr"}, bug.ClosedStatus),
		NewSetStatusOp(bug.Person{Name: "René Descartes", Email: "rene@descartes.fr\u001b"}, bug.ClosedStatus),
		NewSetStatusOp(bug.Person{Name: "René \nDescartes", Email: "rene@descartes.fr"}, bug.ClosedStatus),
		NewSetStatusOp(bug.Person{Name: "René Descartes", Email: "rene@\ndescartes.fr"}, bug.ClosedStatus),
		CreateOperation{OpBase: bug.OpBase{
			Author:        rene,
			UnixTime:      0,
			OperationType: bug.CreateOp,
		},
			Title:   "title",
			Message: "message",
		},

		NewCreateOp(rene, "multi\nline", "message", nil),
		NewCreateOp(rene, "title", "message", []git.Hash{git.Hash("invalid")}),
		NewCreateOp(rene, "title\u001b", "message", nil),
		NewCreateOp(rene, "title", "message\u001b", nil),
		NewSetTitleOp(rene, "multi\nline", "title1"),
		NewSetTitleOp(rene, "title", "multi\nline"),
		NewSetTitleOp(rene, "title\u001b", "title2"),
		NewSetTitleOp(rene, "title", "title2\u001b"),
		NewAddCommentOp(rene, "", nil),
		NewAddCommentOp(rene, "message\u001b", nil),
		NewAddCommentOp(rene, "message", []git.Hash{git.Hash("invalid")}),
		NewSetStatusOp(rene, 1000),
		NewSetStatusOp(rene, 0),
		NewLabelChangeOperation(rene, []bug.Label{}, []bug.Label{}),
		NewLabelChangeOperation(rene, []bug.Label{"multi\nline"}, []bug.Label{}),
	}

	for i, op := range bad {
		if err := op.Validate(); err == nil {
			t.Fatal("validation should have failed", i, op)
		}
	}

}
