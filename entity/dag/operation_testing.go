package dag

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

// SerializeRoundTripTest realize a marshall/unmarshall round-trip in the same condition as with OperationPack,
// and check if the recovered operation is identical.
func SerializeRoundTripTest[OpT Operation](
	t *testing.T,
	unmarshaler OperationUnmarshaler,
	maker func(author identity.Interface, unixTime int64) (OpT, entity.Resolvers),
) {
	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	op, resolvers := maker(rene, time.Now().Unix())
	// enforce having an id
	op.Id()

	data, err := json.Marshal(op)
	require.NoError(t, err)

	after, err := unmarshaler(data, resolvers)
	require.NoError(t, err)

	// Set the id from the serialized data
	after.setId(entity.DeriveId(data))
	// Set the author, as OperationPack would do
	after.setAuthor(rene)

	require.Equal(t, op, after)
}
