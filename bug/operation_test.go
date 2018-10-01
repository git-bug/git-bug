package bug

import (
	"testing"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
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

func TestMetadata(t *testing.T) {
	op := NewCreateOp(rene, unix, "title", "message", nil)

	op.SetMetadata("key", "value")

	val, ok := op.GetMetadata("key")
	require.True(t, ok)
	require.Equal(t, val, "value")
}

func TestHash(t *testing.T) {
	repos := []repository.ClockedRepo{
		repository.NewMockRepoForTest(),
		createRepo(false),
	}

	for _, repo := range repos {
		b, op, err := Create(rene, unix, "title", "message")
		require.Nil(t, err)

		h1, err := op.Hash()
		require.Nil(t, err)

		err = b.Commit(repo)
		require.Nil(t, err)

		op2 := b.FirstOp()

		h2, err := op2.Hash()
		require.Nil(t, err)

		require.Equal(t, h1, h2)

		b2, err := ReadLocalBug(repo, b.id)
		require.Nil(t, err)

		op3 := b2.FirstOp()

		h3, err := op3.Hash()
		require.Nil(t, err)

		require.Equal(t, h1, h3)
	}
}
