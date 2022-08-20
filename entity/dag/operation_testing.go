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
func SerializeRoundTripTest[OpT Operation](t *testing.T, maker func(author identity.Interface, unixTime int64) OpT) {
	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	op := maker(rene, time.Now().Unix())
	// enforce having an id
	op.Id()

	rdt := &roundTripper[OpT]{Before: op, author: rene}

	data, err := json.Marshal(rdt)
	require.NoError(t, err)

	err = json.Unmarshal(data, &rdt)
	require.NoError(t, err)

	require.Equal(t, op, rdt.after)
}

type roundTripper[OpT Operation] struct {
	Before OpT
	author identity.Interface
	after  OpT
}

func (r *roundTripper[OpT]) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Before)
}

func (r *roundTripper[OpT]) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &r.after); err != nil {
		return err
	}
	// Set the id from the serialized data
	r.after.setId(entity.DeriveId(data))
	// Set the author, as OperationPack would do
	r.after.setAuthor(r.author)
	return nil
}
