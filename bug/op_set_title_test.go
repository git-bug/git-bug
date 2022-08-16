package bug

import (
	"testing"

	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/identity"
)

func TestSetTitleSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, func(author identity.Interface, unixTime int64) *SetTitleOperation {
		return NewSetTitleOp(author, unixTime, "title", "was")
	})
}
