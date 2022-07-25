package dag

import (
	"testing"

	"github.com/MichaelMure/git-bug/identity"
)

func TestNoopSerialize(t *testing.T) {
	SerializeRoundTripTest(t, func(author identity.Interface, unixTime int64) *NoOpOperation[*snapshotMock] {
		return NewNoOpOp[*snapshotMock](1, author, unixTime)
	})
}
