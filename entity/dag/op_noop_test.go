package dag

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entity"
)

func TestNoopSerialize(t *testing.T) {
	repo, rene, _, resolvers, def := makeTestContext()
	def.OperationUnmarshaler = func(_ context.Context, raw json.RawMessage, resolver entity.Resolvers) (Operation, error) {
		var op NoOpOperation[*snapshotMock]
		err := json.Unmarshal(raw, &op)
		return &op, err
	}

	before := NewNoOpOp[*snapshotMock](1, rene, time.Now().Unix())

	e := New(def)
	e.Append(before)
	require.NoError(t, e.Commit(t.Context(), repo))

	after, err := Read(t.Context(), def, wrapper, repo, resolvers, e.Id())
	require.NoError(t, err)
	// committed with the edit time of its pack, see Operation.EditTime
	require.Equal(t, e.Operations(), after.Operations())
}
