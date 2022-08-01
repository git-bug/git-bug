package bug

import (
	"testing"

	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/identity"
)

func TestLabelChangeSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, func(author identity.Interface, unixTime int64) *LabelChangeOperation {
		return NewLabelChangeOperation(author, unixTime, []Label{"added"}, []Label{"removed"})
	})
	dag.SerializeRoundTripTest(t, func(author identity.Interface, unixTime int64) *LabelChangeOperation {
		return NewLabelChangeOperation(author, unixTime, []Label{"added"}, nil)
	})
	dag.SerializeRoundTripTest(t, func(author identity.Interface, unixTime int64) *LabelChangeOperation {
		return NewLabelChangeOperation(author, unixTime, nil, []Label{"removed"})
	})
}
