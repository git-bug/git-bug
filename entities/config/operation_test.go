package config

import (
	"encoding/json"
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/common"
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

	t.Run("times", func(t *testing.T) {
		// every operation records times, even with no clock
		op, err := NewCreateOp(repository.NewMockRepoClock(), rene, unix)
		require.NoError(t, err)
		require.NoError(t, op.Validate())
		data, err := json.Marshal(op)
		require.NoError(t, err)
		require.Contains(t, string(data), `"times":{}`)

		create, err := NewCreateOp(clocks, rene, unix)
		require.NoError(t, err)
		description, err := NewRepoSetDescriptionOp(clocks, rene, unix, "description")
		require.NoError(t, err)
		set, err := NewLabelsSetOp(clocks, rene, unix, "bug", LabelsDefinition{})
		require.NoError(t, err)
		remove, err := NewLabelsRemoveOp(clocks, rene, unix, "bug")
		require.NoError(t, err)
		create.Times, description.Times, set.Times, remove.Times = nil, nil, nil, nil
		for _, op := range []Operation{create, description, set, remove} {
			require.ErrorContains(t, op.Validate(), "times", "operation type %d", op.Type())
		}
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

	t.Run("null", func(t *testing.T) {
		// marshal returns the JSON of an operation with these members, on top of
		// the common ones
		marshal := func(t *testing.T, members map[string]any) json.RawMessage {
			t.Helper()
			op := map[string]any{
				"timestamp": unix,
				"nonce":     make([]byte, 20),
				"times":     map[string]lamport.Time{"bugs-edit": 42},
			}
			maps.Copy(op, members)
			raw, err := json.Marshal(op)
			require.NoError(t, err)
			return raw
		}

		t.Run("optional member", func(t *testing.T) {
			// an optional member that is null reads as absent
			for _, members := range []map[string]any{
				{"type": LabelsSetOp, "key": "bug", "fields": nil, "clear": nil},
				{"type": LabelsSetOp, "key": "bug", "fields": map[string]any{"color": nil}},
			} {
				op, err := operationUnmarshaler(t.Context(), marshal(t, members), nil)
				require.NoError(t, err, "%v", members)
				require.Equal(t, LabelsDefinition{}, op.(*LabelsSetOperation).Fields, "%v", members)
			}
		})

		t.Run("required member", func(t *testing.T) {
			// a required member that is null is absent, which makes the operation
			// invalid
			for _, members := range []map[string]any{
				{"type": RepoSetDescriptionOp},
				{"type": RepoSetDescriptionOp, "value": nil},
				{"type": LabelsAllowOp},
				{"type": LabelsAllowOp, "key": nil},
				{"type": LabelsSetOp, "key": nil},
				{"type": LabelsAllowOp, "key": "bug", "times": nil},
			} {
				_, err := operationUnmarshaler(t.Context(), marshal(t, members), nil)
				require.ErrorContains(t, err, "missing member", "%v", members)
			}
		})

		t.Run("unknown member", func(t *testing.T) {
			// a member that the kind doesn't have is an unknown field, null or not
			for _, members := range []map[string]any{
				{"type": CreateOp, "value": nil},
				{"type": LabelsSetOp, "key": "bug", "value": nil},
				{"type": LabelsRemoveOp, "key": "bug", "fields": nil},
			} {
				_, err := operationUnmarshaler(t.Context(), marshal(t, members), nil)
				require.NoError(t, err, "%v", members)
			}
		})

		t.Run("null in a map", func(t *testing.T) {
			_, err := operationUnmarshaler(t.Context(), marshal(t, map[string]any{"type": LabelsAllowOp, "key": "bug", "times": map[string]any{"bugs-edit": nil}}), nil)
			require.ErrorContains(t, err, "null in a map")
		})

		t.Run("zero value", func(t *testing.T) {
			// a set value writes its value, even the zero one, which reads back
			op, err := NewRepoSetDescriptionOp(clocks, rene, unix, "")
			require.NoError(t, err)
			data, err := json.Marshal(op)
			require.NoError(t, err)
			require.Contains(t, string(data), `"value":""`)
			read, err := operationUnmarshaler(t.Context(), data, nil)
			require.NoError(t, err)
			require.Equal(t, "", read.(*RepoSetDescriptionOperation).Value)
		})

		// clear is an array of names
		for _, clear := range []any{"color", []any{42}, []any{nil}} {
			members := map[string]any{"type": LabelsSetOp, "key": "bug", "clear": clear}
			_, err := operationUnmarshaler(t.Context(), marshal(t, members), nil)
			require.Error(t, err, "%v", members)
		}
	})

	t.Run("clear", func(t *testing.T) {
		dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*LabelsSetOperation, entity.Resolvers) {
			op, err := NewLabelsSetOp(clocks, author, unixTime, "crash", LabelsDefinition{Color: SetField("#ff9800"), Description: ClearField[string]()})
			require.NoError(t, err)
			return op, nil
		})

		// the set entry examples of testdata/config.json in the spec: the
		// definition of crash after one operation, from the same one each time
		before := LabelsDefinition{Color: SetField("#f44336"), Description: SetField("Program dies")}
		for _, c := range []struct {
			members string
			after   LabelsDefinition
		}{
			{`"key": "crash"`, before},
			{`"key": "crash", "fields": {"color": "#ff9800"}`, LabelsDefinition{Color: SetField("#ff9800"), Description: SetField("Program dies")}},
			{`"key": "crash", "clear": ["description"]`, LabelsDefinition{Color: SetField("#f44336")}},
			{`"key": "crash", "fields": {"color": "#ff9800"}, "clear": ["description"]`, LabelsDefinition{Color: SetField("#ff9800")}},
			{`"key": "crash", "fields": {"description": null}`, before},
			{`"key": "crash", "fields": {"color": "#ff9800"}, "clear": ["color"]`, LabelsDefinition{Color: SetField("#ff9800"), Description: SetField("Program dies")}},
			{`"key": "crash", "clear": ["color", "description"]`, LabelsDefinition{}},
			{`"key": "crash", "clear": ["priority"]`, before},
		} {
			raw := `{"type": 20001, "timestamp": 1609459200, "nonce": "AAAAAAAAAAAAAAAAAAAAAAAAAAAA", "times": {"bugs-edit": 137}, ` + c.members + `}`
			op, err := operationUnmarshaler(t.Context(), json.RawMessage(raw), nil)
			require.NoError(t, err, c.members)
			snapshot := &Snapshot{}
			snapshot.Labels.definitions = map_[common.Label, LabelsDefinition]{"crash": before}
			op.(*LabelsSetOperation).Apply(snapshot)
			after, ok := snapshot.Labels.Get("crash")
			require.True(t, ok, c.members)
			require.Equal(t, c.after, after, c.members)
		}
	})
}
