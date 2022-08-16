package bug

import (
	"testing"

	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/identity"
)

func TestSetStatusSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, func(author identity.Interface, unixTime int64) *SetStatusOperation {
		return NewSetStatusOp(author, unixTime, ClosedStatus)
	})
}
