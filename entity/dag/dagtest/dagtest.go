// Package dagtest provides test helpers for the entities built on entity/dag.
package dagtest

import (
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

// SerializeRoundTripTest stores the operation made by maker in an entity,
// reads that entity back, and checks that the recovered operation is
// identical.
func SerializeRoundTripTest[OpT dag.Operation](
	t *testing.T,
	unmarshaler dag.OperationUnmarshaler,
	maker func(author identity.Interface, unixTime int64) (OpT, entity.Resolvers),
) {
	t.Helper()

	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	require.NoError(t, rene.Commit(repo))

	before, resolvers := maker(rene, time.Now().Unix())

	resolvers = maps.Clone(resolvers)
	if resolvers == nil {
		resolvers = entity.Resolvers{}
	}
	resolvers[&identity.Identity{}] = entity.MakeResolver(rene)

	def := dag.Definition{
		Typename:             "roundtrip",
		Namespace:            "roundtrip",
		OperationUnmarshaler: unmarshaler,
		FormatVersion:        1,
	}

	e := dag.New(def)
	e.Append(before)
	require.NoError(t, e.Commit(repo))

	read, err := dag.Read(t.Context(), def, func(e *dag.Entity) *dag.Entity { return e }, repo, resolvers, e.Id())
	require.NoError(t, err)
	require.Len(t, read.Operations(), 1)

	require.Equal(t, before, read.Operations()[0])
}
