package bug

import (
	"testing"

	"github.com/MichaelMure/git-bug/entities/common"
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

func TestSetStatusSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, func(author identity.Interface, unixTime int64) *SetStatusOperation {
		return NewSetStatusOp(author, unixTime, common.ClosedStatus)
	})
}
