package dag

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entity"
)

func makeUnknownOpRaw(t *testing.T, opType OperationType, extra ...map[string]string) json.RawMessage {
	t.Helper()
	m := map[string]interface{}{
		"type":      opType,
		"timestamp": time.Now().Unix(),
		"nonce":     make([]byte, 20),
	}
	for _, e := range extra {
		for k, v := range e {
			m[k] = v
		}
	}
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	return raw
}

func TestNewUnknownOpPreservesRawJSON(t *testing.T) {
	raw := makeUnknownOpRaw(t, OperationType(999))
	op, err := NewUnknownOp[*snapshotMock](raw)
	require.NoError(t, err)
	require.Equal(t, raw, op.RawJSON)
	// the id a read would set
	require.Equal(t, entity.DeriveId(raw), op.Id())
}

func TestUnknownOpType(t *testing.T) {
	raw := makeUnknownOpRaw(t, OperationType(999))
	op, err := NewUnknownOp[*snapshotMock](raw)
	require.NoError(t, err)
	require.Equal(t, OperationType(999), op.Type())
}

func TestUnknownOpMarshalJSONRoundTrip(t *testing.T) {
	raw := makeUnknownOpRaw(t, OperationType(999), map[string]string{"extra_field": "preserved"})
	op, err := NewUnknownOp[*snapshotMock](raw)
	require.NoError(t, err)

	marshaled, err := json.Marshal(op)
	require.NoError(t, err)
	require.JSONEq(t, string(raw), string(marshaled))
}

func TestUnknownOpValidate(t *testing.T) {
	raw := makeUnknownOpRaw(t, OperationType(999))
	op, err := NewUnknownOp[*snapshotMock](raw)
	require.NoError(t, err)
	require.NoError(t, op.Validate())
}

func TestUnknownOpApplyIsNoop(t *testing.T) {
	raw := makeUnknownOpRaw(t, OperationType(999))
	op, err := NewUnknownOp[*snapshotMock](raw)
	require.NoError(t, err)

	snap := &snapshotMock{}
	op.Apply(snap)
	require.Empty(t, snap.ops)
}

func TestUnknownOpId(t *testing.T) {
	raw := makeUnknownOpRaw(t, OperationType(999))
	op, err := NewUnknownOp[*snapshotMock](raw)
	require.NoError(t, err)

	// Simulate the id assignment done by unmarshallPack after calling the unmarshaler.
	op.setId(entity.DeriveId(raw))

	id := op.Id()
	require.NoError(t, id.Validate())
	// Id is stable across calls.
	require.Equal(t, id, op.Id())
}

func TestNewUnknownOpInvalidJSON(t *testing.T) {
	_, err := NewUnknownOp[*snapshotMock](json.RawMessage(`not-valid-json`))
	require.Error(t, err)
}
