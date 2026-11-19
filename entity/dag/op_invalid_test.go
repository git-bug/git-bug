package dag

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entity"
)

func TestNewInvalidOp(t *testing.T) {
	raw := makeUnknownOpRaw(t, Op1, map[string]string{"field_1": "foo"})
	reason := errors.New("can't be decoded")

	op, err := NewInvalidOp[*snapshotMock](raw, reason)
	require.NoError(t, err)

	// kept verbatim, with the id a read would set
	require.Equal(t, raw, op.RawJSON)
	require.Equal(t, Op1, op.Type())
	require.Equal(t, entity.DeriveId(raw), op.Id())
	marshaled, err := json.Marshal(op)
	require.NoError(t, err)
	require.JSONEq(t, string(raw), string(marshaled))

	// it is flagged from the start, and doesn't validate
	require.ErrorIs(t, op.Invalid(), reason)
	require.ErrorIs(t, op.Validate(), reason)

	// it has no effect
	snap := &snapshotMock{}
	op.Apply(snap)
	require.Equal(t, &snapshotMock{}, snap)
}
