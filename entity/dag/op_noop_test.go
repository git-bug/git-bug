package dag

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
)

func TestNoopSerialize(t *testing.T) {
	before, after, err := SerializeRoundTrip(func(raw json.RawMessage, resolver entity.Resolvers) (Operation, error) {
		var op NoOpOperation[*snapshotMock]
		err := json.Unmarshal(raw, &op)
		return &op, err
	}, func(author identity.Interface, unixTime int64) (*NoOpOperation[*snapshotMock], entity.Resolvers) {
		return NewNoOpOp[*snapshotMock](1, author, unixTime), nil
	})
	require.NoError(t, err)
	require.Equal(t, before, after)
}
