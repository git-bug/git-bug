package dag

import (
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/repository"

	"github.com/stretchr/testify/require"
)

type snapshotMock struct {
	ops []Operation
}

func (s *snapshotMock) AllOperations() []Operation {
	return s.ops
}

func TestSetMetadata(t *testing.T) {
	snap := &snapshotMock{}

	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	unix := time.Now().Unix()

	target1 := NewNoOpOp[*snapshotMock](1, rene, unix)
	target1.SetMetadata("key", "value")
	snap.ops = append(snap.ops, target1)

	target2 := NewNoOpOp[*snapshotMock](1, rene, unix)
	target2.SetMetadata("key2", "value2")
	snap.ops = append(snap.ops, target2)

	op1 := NewSetMetadataOp[*snapshotMock](2, rene, unix, target1.Id(), map[string]string{
		"key":  "override",
		"key2": "value",
	})

	op1.Apply(snap)
	snap.ops = append(snap.ops, op1)

	target1Metadata := snap.AllOperations()[0].AllMetadata()
	require.Len(t, target1Metadata, 2)
	// original key is not overrided
	require.Equal(t, target1Metadata["key"], "value")
	// new key is set
	require.Equal(t, target1Metadata["key2"], "value")

	target2Metadata := snap.AllOperations()[1].AllMetadata()
	require.Len(t, target2Metadata, 1)
	require.Equal(t, target2Metadata["key2"], "value2")

	op2 := NewSetMetadataOp[*snapshotMock](2, rene, unix, target2.Id(), map[string]string{
		"key2": "value",
		"key3": "value3",
	})

	op2.Apply(snap)
	snap.ops = append(snap.ops, op2)

	target1Metadata = snap.AllOperations()[0].AllMetadata()
	require.Len(t, target1Metadata, 2)
	require.Equal(t, target1Metadata["key"], "value")
	require.Equal(t, target1Metadata["key2"], "value")

	target2Metadata = snap.AllOperations()[1].AllMetadata()
	require.Len(t, target2Metadata, 2)
	// original key is not overrided
	require.Equal(t, target2Metadata["key2"], "value2")
	// new key is set
	require.Equal(t, target2Metadata["key3"], "value3")

	op3 := NewSetMetadataOp[*snapshotMock](2, rene, unix, target1.Id(), map[string]string{
		"key":  "override",
		"key2": "override",
	})

	op3.Apply(snap)
	snap.ops = append(snap.ops, op3)

	target1Metadata = snap.AllOperations()[0].AllMetadata()
	require.Len(t, target1Metadata, 2)
	// original key is not overrided
	require.Equal(t, target1Metadata["key"], "value")
	// previously set key is not overrided
	require.Equal(t, target1Metadata["key2"], "value")

	target2Metadata = snap.AllOperations()[1].AllMetadata()
	require.Len(t, target2Metadata, 2)
	require.Equal(t, target2Metadata["key2"], "value2")
	require.Equal(t, target2Metadata["key3"], "value3")
}

func TestSetMetadataSerialize(t *testing.T) {
	SerializeRoundTripTest(t, func(author identity.Interface, unixTime int64) *SetMetadataOperation[*snapshotMock] {
		return NewSetMetadataOp[*snapshotMock](1, author, unixTime, "message", map[string]string{
			"key1": "value1",
			"key2": "value2",
		})
	})
}
