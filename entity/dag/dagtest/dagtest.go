// Package dagtest provides test helpers for the entities built on entity/dag.
package dagtest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

// SerializeRoundTripTest realize a marshall/unmarshall round-trip in the same
// condition as with OperationPack, and check if the recovered operation is
// identical.
func SerializeRoundTripTest[OpT dag.Operation](
	t *testing.T,
	unmarshaler dag.OperationUnmarshaler,
	maker func(author identity.Interface, unixTime int64) (OpT, entity.Resolvers),
) {
	t.Helper()

	before, after, err := dag.SerializeRoundTrip(unmarshaler, maker)
	require.NoError(t, err)
	require.Equal(t, before, after)
}
