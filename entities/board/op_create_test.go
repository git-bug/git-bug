package board

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

func TestCreate(t *testing.T) {
	snap := Snapshot{}

	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	unix := time.Now().Unix()

	create := NewCreateOp(rene, unix, "title", "description", DefaultColumns)
	create.Apply(&snap)

	id := create.Id()
	require.NoError(t, id.Validate())

	require.Equal(t, id, snap.Id())
	require.Equal(t, "title", snap.Title)
	require.Equal(t, "description", snap.Description)
	require.Len(t, snap.Columns, len(DefaultColumns))
	for i, column := range DefaultColumns {
		require.Equal(t, column, snap.Columns[i].Name)
	}

	// Make sure an extra Create Op doesn't mess things
	isaac, err := identity.NewIdentity(repo, "Isaac Newton", "isaac@newton.uk")
	require.NoError(t, err)
	create2 := NewCreateOp(isaac, unix, "title2", "description2", DefaultColumns)
	create2.Apply(&snap)

	require.Equal(t, id, snap.Id())
	require.Equal(t, "title", snap.Title)
	require.Equal(t, "description", snap.Description)
	require.Len(t, snap.Columns, len(DefaultColumns))
	for i, column := range DefaultColumns {
		require.Equal(t, column, snap.Columns[i].Name)
	}
}

func TestNonUnique(t *testing.T) {
	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	unix := time.Now().Unix()

	create := NewCreateOp(rene, unix, "title", "description", []string{
		"foo", "bar", "foo",
	})

	require.Error(t, create.Validate())
}

func TestCreateSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*CreateOperation, entity.Resolvers) {
		return NewCreateOp(author, unixTime, "title", "description", DefaultColumns), nil
	})
}
