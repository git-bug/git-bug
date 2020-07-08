package bug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

func TestValidate(t *testing.T) {
	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
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
		NewSetStatusOp(identity.NewIdentity("", "rene@descartes.fr"), unix, ClosedStatus),
		NewSetStatusOp(identity.NewIdentity("René Descartes\u001b", "rene@descartes.fr"), unix, ClosedStatus),
		NewSetStatusOp(identity.NewIdentity("René Descartes", "rene@descartes.fr\u001b"), unix, ClosedStatus),
		NewSetStatusOp(identity.NewIdentity("René \nDescartes", "rene@descartes.fr"), unix, ClosedStatus),
		NewSetStatusOp(identity.NewIdentity("René Descartes", "rene@\ndescartes.fr"), unix, ClosedStatus),
		&CreateOperation{OpBase: OpBase{
			Author:        rene,
			UnixTime:      0,
			OperationType: CreateOp,
		},
			Title:   "title",
			Message: "message",
		},

		NewCreateOp(rene, unix, "multi\nline", "message", nil),
		NewCreateOp(rene, unix, "title", "message", []repository.Hash{repository.Hash("invalid")}),
		NewCreateOp(rene, unix, "title\u001b", "message", nil),
		NewCreateOp(rene, unix, "title", "message\u001b", nil),
		NewSetTitleOp(rene, unix, "multi\nline", "title1"),
		NewSetTitleOp(rene, unix, "title", "multi\nline"),
		NewSetTitleOp(rene, unix, "title\u001b", "title2"),
		NewSetTitleOp(rene, unix, "title", "title2\u001b"),
		NewAddCommentOp(rene, unix, "message\u001b", nil),
		NewAddCommentOp(rene, unix, "message", []repository.Hash{repository.Hash("invalid")}),
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
	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	op := NewCreateOp(rene, time.Now().Unix(), "title", "message", nil)

	op.SetMetadata("key", "value")

	val, ok := op.GetMetadata("key")
	require.True(t, ok)
	require.Equal(t, val, "value")
}

func TestID(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(repo)

	repos := []repository.ClockedRepo{
		repository.NewMockRepoForTest(),
		repo,
	}

	for _, repo := range repos {
		rene := identity.NewBare("René Descartes", "rene@descartes.fr")

		b, op, err := Create(rene, time.Now().Unix(), "title", "message")
		require.Nil(t, err)

		id1 := op.Id()
		require.NoError(t, id1.Validate())

		err = b.Commit(repo)
		require.Nil(t, err)

		op2 := b.FirstOp()

		id2 := op2.Id()
		require.NoError(t, id2.Validate())

		require.Equal(t, id1, id2)

		b2, err := ReadLocalBug(repo, b.Id())
		require.Nil(t, err)

		op3 := b2.FirstOp()

		id3 := op3.Id()
		require.NoError(t, id3.Validate())

		require.Equal(t, id1, id3)
	}
}
