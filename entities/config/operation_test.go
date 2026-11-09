package config

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/entity/dag/dagtest"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

// makeTestContext returns clocks with a time each, recorded by every operation,
// and two identities.
func makeTestContext(t *testing.T) (repository.RepoClock, identity.Interface, identity.Interface) {
	t.Helper()

	clocks := repository.NewMockRepoClock()
	for _, name := range []string{"bugs-create", "bugs-edit"} {
		_, err := clocks.GetOrCreateClock(name, 42)
		require.NoError(t, err)
	}

	rene, err := identity.NewIdentity(clocks, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	isaac, err := identity.NewIdentity(clocks, "Isaac Newton", "isaac@newton.uk")
	require.NoError(t, err)

	return clocks, rene, isaac
}

func TestOperations(t *testing.T) {
	clocks, rene, _ := makeTestContext(t)
	unix := time.Now().Unix()

	t.Run("numbered by section", func(t *testing.T) {
		for number, operations := range sectionOperations {
			require.NotEmpty(t, operations)
			for opType, newOp := range operations {
				require.Equal(t, number, int(opType)/sectionSize, "operation type %d", opType)
				require.NotZero(t, int(opType)%sectionSize, "operation type %d", opType)
				require.NotNil(t, newOp())
			}
		}
	})

	t.Run("create", func(t *testing.T) {
		op, err := NewCreateOp(clocks, rene, unix)
		require.NoError(t, err)
		require.NoError(t, op.Validate())
		require.Equal(t, map[string]lamport.Time{"bugs-create": 42, "bugs-edit": 42}, op.Times)

		op, err = NewCreateOp(clocks, rene, 0)
		require.NoError(t, err)
		require.Error(t, op.Validate())

		dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*CreateOperation, entity.Resolvers) {
			op, err := NewCreateOp(clocks, author, unixTime)
			require.NoError(t, err)
			return op, nil
		})
	})

	t.Run("unknown", func(t *testing.T) {
		// An operation type that this client doesn't know is kept as it is, in a
		// known section or not
		for _, opType := range []dag.OperationType{labelsSection*sectionSize + 99, 999} {
			raw, err := json.Marshal(struct {
				Type      dag.OperationType `json:"type"`
				Timestamp int64             `json:"timestamp"`
				Nonce     []byte            `json:"nonce"`
				Extra     string            `json:"future_field"`
			}{
				Type:      opType,
				Timestamp: unix,
				Nonce:     make([]byte, 20),
				Extra:     "preserved",
			})
			require.NoError(t, err)

			op, err := operationUnmarshaler(t.Context(), raw, nil)
			require.NoError(t, err)
			require.IsType(t, &dag.UnknownOperation[*Snapshot]{}, op)

			marshaled, err := json.Marshal(op)
			require.NoError(t, err)
			require.JSONEq(t, string(raw), string(marshaled))
		}
	})
}
