package dag

import (
	"encoding/json"
	"time"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

// SerializeRoundTrip marshals the operation made by maker and unmarshals it
// back, in the same conditions as OperationPack. It returns both, for tests to
// check that they are identical.
func SerializeRoundTrip[OpT Operation](
	unmarshaler OperationUnmarshaler,
	maker func(author identity.Interface, unixTime int64) (OpT, entity.Resolvers),
) (before OpT, after Operation, err error) {
	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	if err != nil {
		return before, nil, err
	}

	before, resolvers := maker(rene, time.Now().Unix())
	// enforce having an id
	before.Id()

	data, err := json.Marshal(before)
	if err != nil {
		return before, nil, err
	}

	after, err = unmarshaler(data, resolvers)
	if err != nil {
		return before, nil, err
	}

	// Set the id from the serialized data
	after.setId(entity.DeriveId(data))
	// Set the author, as OperationPack would do
	after.setAuthor(rene)

	return before, after, nil
}
